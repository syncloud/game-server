package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/syncloud/games/backend/auth"
	"github.com/syncloud/games/backend/catalog"
	"github.com/syncloud/games/backend/db"
	"github.com/syncloud/games/backend/installer"
	"github.com/syncloud/games/backend/query"
	"github.com/syncloud/games/backend/runner"
	"github.com/syncloud/games/backend/server"
	"github.com/syncloud/games/backend/steam"
)

const oidcConfigPath = "/var/snap/games/current/oidc.json"

type Game = catalog.Game

const (
	socketPath    = "/var/snap/games/current/backend.sock"
	cliSocketPath = "/var/snap/games/current/cli.sock"
	dbPath        = "/var/snap/games/current/database.db"
)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func gameByID(id string) *Game {
	g, ok := catalog.Get(id)
	if !ok {
		return nil
	}
	return &g
}

func main() {
	logger := log.New(os.Stdout, "backend: ", log.LstdFlags)

	if err := catalog.Start(); err != nil {
		logger.Fatalf("catalog: %v", err)
	}

	store, err := openDB(logger)
	if err != nil {
		logger.Fatalf("db: %v", err)
	}
	defer store.Close()
	run := runner.New(logger)
	inst := installer.New(
		installer.ServersBaseDir,
		installer.NewSteamInstaller(installer.SteamCMDPath, installer.SteamLib32, installer.SteamLib64),
		installer.NewEggInstaller(installer.JREBinDir),
	)

	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		logger.Fatalf("listen: %v", err)
	}
	if err := os.Chmod(socketPath, 0666); err != nil {
		logger.Fatalf("chmod: %v", err)
	}

	_ = os.Remove(cliSocketPath)
	cliListener, err := net.Listen("unix", cliSocketPath)
	if err != nil {
		logger.Fatalf("listen cli: %v", err)
	}
	if err := os.Chmod(cliSocketPath, 0660); err != nil {
		logger.Fatalf("chmod cli: %v", err)
	}

	authSvc := loadAuth(logger)

	api := http.NewServeMux()
	api.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	api.HandleFunc("/api/v1/me", func(w http.ResponseWriter, r *http.Request) {
		if authSvc != nil {
			authSvc.HandleMe(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"sub": "unknown"})
	})
	api.HandleFunc("/api/v1/games", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, catalog.All())
	})
	api.HandleFunc("/api/v1/catalog/sources", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, catalog.Sources())
	})
	api.HandleFunc("/api/v1/steam/login", func(w http.ResponseWriter, r *http.Request) {
		handleSteamLogin(w, r)
	})
	api.HandleFunc("/api/v1/steam/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"linked":   steam.StoredUsername() != "",
			"username": steam.StoredUsername(),
		})
	})
	api.HandleFunc("/api/v1/servers", func(w http.ResponseWriter, r *http.Request) {
		handleServers(w, r, store)
	})
	api.HandleFunc("/api/v1/servers/", func(w http.ResponseWriter, r *http.Request) {
		handleServerByID(w, r, store, run, inst)
	})

	mux := http.NewServeMux()
	if authSvc != nil {
		mux.HandleFunc("/auth/login", authSvc.HandleLogin)
		mux.HandleFunc("/auth/callback", authSvc.HandleCallback)
		mux.HandleFunc("/auth/logout", authSvc.HandleLogout)
		mux.Handle("/api/", authSvc.Middleware(api))
	} else {
		logger.Printf("auth disabled — OIDC config not loaded; /api/ unprotected (dev mode)")
		mux.Handle("/api/", api)
	}

	cliMux := http.NewServeMux()
	cliMux.Handle("/api/", api)

	go func() {
		logger.Printf("listening on %s (cli)", cliSocketPath)
		if err := http.Serve(cliListener, cliMux); err != nil {
			logger.Fatalf("serve cli: %v", err)
		}
	}()

	logger.Printf("listening on %s", socketPath)
	if err := http.Serve(listener, mux); err != nil {
		logger.Fatalf("serve: %v", err)
	}
}

type oidcFileConfig struct {
	AuthUrl      string `json:"authUrl"`
	AuthSocket   string `json:"authSocket"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	RedirectUrl  string `json:"redirectUrl"`
}

func loadAuth(logger *log.Logger) *auth.Service {
	data, err := os.ReadFile(oidcConfigPath)
	if err != nil {
		logger.Printf("auth: oidc.json missing (%v); /api/ will be unprotected", err)
		return nil
	}
	var c oidcFileConfig
	if err := json.Unmarshal(data, &c); err != nil {
		logger.Printf("auth: oidc.json parse: %v", err)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if c.AuthSocket == "" {
		logger.Printf("auth: oidc.json missing authSocket; /api/ will be unprotected")
		return nil
	}
	svc, err := auth.NewService(ctx, logger, c.AuthUrl, c.AuthSocket, c.ClientID, c.ClientSecret, c.ClientSecret, c.RedirectUrl)
	if err != nil {
		logger.Printf("auth: init: %v", err)
		return nil
	}
	logger.Printf("auth: OIDC ready (provider=%s socket=%s client=%s redirect=%s)", c.AuthUrl, c.AuthSocket, c.ClientID, c.RedirectUrl)
	return svc
}

func openDB(logger *log.Logger) (*db.DB, error) {
	d := db.New(dbPath)
	if err := d.Start(); err != nil {
		if _, statErr := os.Stat("/var/snap/games/current"); statErr != nil {
			logger.Printf("data dir missing, falling back to in-memory db: %v", statErr)
			d = db.New(":memory:")
			if err := d.Start(); err != nil {
				return nil, err
			}
			return d, nil
		}
		return nil, err
	}
	return d, nil
}

type createRequest struct {
	Name     string `json:"name"`
	GameID   string `json:"gameId"`
	Port     int    `json:"port"`
	StartCmd string `json:"startCmd"`
}

func handleServers(w http.ResponseWriter, r *http.Request, store *db.DB) {
	switch r.Method {
	case http.MethodGet:
		list, err := store.ListServers()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var req createRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "name required")
			return
		}
		game := gameByID(req.GameID)
		if game == nil {
			writeError(w, http.StatusBadRequest, "unknown gameId")
			return
		}
		if req.Port == 0 {
			req.Port = game.DefaultPort
		}
		s, err := store.CreateServer(server.Server{
			Name:     req.Name,
			GameID:   req.GameID,
			Port:     req.Port,
			StartCmd: req.StartCmd,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, s)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleServerByID(w http.ResponseWriter, r *http.Request, store *db.DB, run *runner.Runner, inst *installer.Installer) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
	parts := strings.SplitN(rest, "/", 2)
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if len(parts) == 2 && parts[1] != "" {
		switch parts[1] {
		case "logs":
			handleLogs(w, r, run, id)
			return
		case "query":
			handleQuery(w, r, store, id)
			return
		default:
			handleServerAction(w, r, store, run, inst, id, parts[1])
			return
		}
	}
	switch r.Method {
	case http.MethodGet:
		s, err := store.GetServer(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if s == nil {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		s.Status = currentStatus(s, run)
		writeJSON(w, http.StatusOK, s)
	case http.MethodDelete:
		_ = run.Stop(id)
		if err := store.DeleteServer(id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleServerAction(w http.ResponseWriter, r *http.Request, store *db.DB, run *runner.Runner, inst *installer.Installer, id int64, action string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s, err := store.GetServer(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if s == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	switch action {
	case "install":
		game := gameByID(s.GameID)
		if game == nil {
			writeError(w, http.StatusBadRequest, "unknown gameId on server")
			return
		}
		_ = store.UpdateServerStatus(id, "installing")
		go runInstall(log.Default(), store, inst, id, *game)
		s.Status = "installing"
	case "start":
		if err := run.Start(id, s.StartCmd, s.InstallDir); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		_ = store.UpdateServerStatus(id, "running")
	case "stop":
		if err := run.Stop(id); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		_ = store.UpdateServerStatus(id, "stopped")
	case "restart":
		if err := run.Restart(id, s.StartCmd, s.InstallDir); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		_ = store.UpdateServerStatus(id, "running")
	default:
		writeError(w, http.StatusNotFound, "unknown action")
		return
	}
	s, _ = store.GetServer(id)
	s.Status = currentStatus(s, run)
	writeJSON(w, http.StatusOK, s)
}

func handleLogs(w http.ResponseWriter, r *http.Request, run *runner.Runner, id int64) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": run.Logs(id)})
}

func handleQuery(w http.ResponseWriter, r *http.Request, store *db.DB, id int64) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s, err := store.GetServer(id)
	if err != nil || s == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	addr := fmt.Sprintf("127.0.0.1:%d", s.Port)
	info, err := query.QueryInfo(addr, 2*time.Second)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

type steamLoginRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	GuardCode string `json:"guardCode"`
}

func handleSteamLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req steamLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	res, err := steam.Login(ctx, req.Username, req.Password, req.GuardCode)
	if res != nil && res.Needs2FA {
		writeJSON(w, http.StatusOK, map[string]any{
			"needsGuard": true,
			"prompt":     res.Prompt,
		})
		return
	}
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"linked":   true,
		"username": res.Username,
	})
}

func currentStatus(s *server.Server, run *runner.Runner) string {
	if s.Status == "installing" || s.Status == "install-error" {
		return s.Status
	}
	if run.Running(s.ID) {
		return "running"
	}
	return "stopped"
}

func runInstall(logger *log.Logger, store *db.DB, inst *installer.Installer, id int64, g Game) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	s, err := store.GetServer(id)
	if err != nil || s == nil {
		return
	}
	steamUser := steam.StoredUsername()
	logger.Printf("install[%d] starting: game=%s source=%s appid=%d steamUser=%q", id, g.ID, g.Source, g.SteamAppID, steamUser)
	result, err := inst.Install(ctx, installer.Game{
		ID:          g.ID,
		Name:        g.Name,
		Source:      g.Source,
		SteamAppID:  g.SteamAppID,
		EggURL:      g.EggURL,
		DefaultPort: g.DefaultPort,
	}, s.Name, s.Port, steamUser, "")
	if err != nil {
		logger.Printf("install[%d] FAILED: %v", id, err)
		_ = store.UpdateServerLastError(id, err.Error())
		_ = store.UpdateServerStatus(id, "install-error")
		return
	}
	logger.Printf("install[%d] OK: dir=%s start=%q", id, result.InstallDir, result.StartCmd)
	_ = store.UpdateServerInstall(id, result.InstallDir, result.StartCmd)
	_ = store.UpdateServerStatus(id, "stopped")
}
