package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/syncloud/games/backend/auth"
	"github.com/syncloud/games/backend/catalog"
	"github.com/syncloud/games/backend/db"
	"github.com/syncloud/games/backend/installer"
	"github.com/syncloud/games/backend/query"
	"github.com/syncloud/games/backend/runner"
	"github.com/syncloud/games/backend/server"
	"github.com/syncloud/games/backend/steam"
)

const (
	socketPath    = "/var/snap/games/current/backend.sock"
	cliSocketPath = "/var/snap/games/current/cli.sock"
)

type Game = catalog.Game

type Api struct {
	logger *zap.Logger
	store  *db.DB
	run    *runner.Runner
	inst   *installer.Installer
	auth   *auth.Service
}

func New(logger *zap.Logger, store *db.DB, run *runner.Runner, inst *installer.Installer, authService *auth.Service) *Api {
	return &Api{logger: logger, store: store, run: run, inst: inst, auth: authService}
}

func (a *Api) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/me", func(w http.ResponseWriter, r *http.Request) {
		if a.auth != nil {
			a.auth.HandleMe(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"sub": "unknown"})
	})
	mux.HandleFunc("/api/v1/games", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, catalog.All())
	})
	mux.HandleFunc("/api/v1/catalog/sources", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, catalog.Sources())
	})
	mux.HandleFunc("/api/v1/steam/login", a.handleSteamLogin)
	mux.HandleFunc("/api/v1/steam/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"linked":   steam.StoredUsername() != "",
			"username": steam.StoredUsername(),
		})
	})
	mux.HandleFunc("/api/v1/servers", a.handleServers)
	mux.HandleFunc("/api/v1/servers/", a.handleServerByID)
	return mux
}

func (a *Api) Start() error {
	apiHandler := a.routes()

	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if err := os.Chmod(socketPath, 0666); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	_ = os.Remove(cliSocketPath)
	cliListener, err := net.Listen("unix", cliSocketPath)
	if err != nil {
		return fmt.Errorf("listen cli: %w", err)
	}
	if err := os.Chmod(cliSocketPath, 0660); err != nil {
		return fmt.Errorf("chmod cli: %w", err)
	}

	mux := http.NewServeMux()
	if a.auth != nil {
		mux.HandleFunc("/auth/login", a.auth.HandleLogin)
		mux.HandleFunc("/auth/callback", a.auth.HandleCallback)
		mux.HandleFunc("/auth/logout", a.auth.HandleLogout)
		mux.Handle("/api/", a.auth.Middleware(apiHandler))
	} else {
		a.logger.Info("auth disabled — OIDC config not loaded; /api/ unprotected (dev mode)")
		mux.Handle("/api/", apiHandler)
	}

	cliMux := http.NewServeMux()
	cliMux.Handle("/api/", apiHandler)

	go func() {
		a.logger.Info("listening (cli)", zap.String("socket", cliSocketPath))
		if err := http.Serve(cliListener, cliMux); err != nil {
			a.logger.Fatal("serve cli", zap.Error(err))
		}
	}()

	a.logger.Info("listening", zap.String("socket", socketPath))
	return http.Serve(listener, mux)
}

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

func hostIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			return ip4.String()
		}
	}
	return ""
}

func enrichServer(s *server.Server, ip string) {
	if s == nil {
		return
	}
	if g, ok := catalog.Get(s.GameID); ok {
		s.GameName = g.Name
	}
	s.LocalIp = ip
}

type createRequest struct {
	Name     string `json:"name"`
	GameID   string `json:"gameId"`
	Port     int    `json:"port"`
	StartCmd string `json:"startCmd"`
}

func (a *Api) handleServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := a.store.ListServers()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		ip := hostIP()
		for i := range list {
			enrichServer(&list[i], ip)
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var req createRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		game := gameByID(req.GameID)
		if game == nil {
			writeError(w, http.StatusBadRequest, "unknown gameId")
			return
		}
		existing, err := a.store.ListServers()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, e := range existing {
			if e.GameID == req.GameID {
				writeError(w, http.StatusConflict, fmt.Sprintf("%s is already installed", game.Name))
				return
			}
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = req.GameID
		}
		if req.Port == 0 {
			req.Port = game.DefaultPort
		}
		s, err := a.store.CreateServer(server.Server{
			Name:     name,
			GameID:   req.GameID,
			Port:     req.Port,
			StartCmd: req.StartCmd,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		enrichServer(s, hostIP())
		writeJSON(w, http.StatusCreated, s)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *Api) handleServerByID(w http.ResponseWriter, r *http.Request) {
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
			a.handleLogs(w, r, id)
			return
		case "query":
			a.handleQuery(w, r, id)
			return
		default:
			a.handleServerAction(w, r, id, parts[1])
			return
		}
	}
	switch r.Method {
	case http.MethodGet:
		s, err := a.store.GetServer(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if s == nil {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		s.Status = a.currentStatus(s)
		enrichServer(s, hostIP())
		writeJSON(w, http.StatusOK, s)
	case http.MethodDelete:
		_ = a.run.Stop(id)
		if err := a.store.DeleteServer(id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *Api) handleServerAction(w http.ResponseWriter, r *http.Request, id int64, action string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s, err := a.store.GetServer(id)
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
		if game.Tier == "disabled" {
			reason := game.DisabledReason
			if reason == "" {
				reason = "game is disabled"
			}
			writeError(w, http.StatusConflict, fmt.Sprintf("game %s is disabled: %s", game.ID, reason))
			return
		}
		_ = a.store.UpdateServerStatus(id, "installing")
		go a.runInstall(id, *game)
		s.Status = "installing"
	case "start":
		if err := a.run.Start(id, s.StartCmd, s.InstallDir); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		_ = a.store.UpdateServerStatus(id, "running")
	case "stop":
		if err := a.run.Stop(id); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		_ = a.store.UpdateServerStatus(id, "stopped")
	case "restart":
		if err := a.run.Restart(id, s.StartCmd, s.InstallDir); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		_ = a.store.UpdateServerStatus(id, "running")
	default:
		writeError(w, http.StatusNotFound, "unknown action")
		return
	}
	s, _ = a.store.GetServer(id)
	s.Status = a.currentStatus(s)
	enrichServer(s, hostIP())
	writeJSON(w, http.StatusOK, s)
}

func (a *Api) handleLogs(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": a.run.Logs(id)})
}

func (a *Api) handleQuery(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s, err := a.store.GetServer(id)
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

func (a *Api) handleSteamLogin(w http.ResponseWriter, r *http.Request) {
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

func (a *Api) currentStatus(s *server.Server) string {
	if s.Status == "installing" || s.Status == "install-error" {
		return s.Status
	}
	if a.run.Running(s.ID) {
		return "running"
	}
	return "stopped"
}

func (a *Api) runInstall(id int64, g Game) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	s, err := a.store.GetServer(id)
	if err != nil || s == nil {
		return
	}
	steamUser := steam.StoredUsername()
	appid := 0
	if g.InstallRecipe != nil {
		appid = g.InstallRecipe.SteamAppID
	}
	a.logger.Info("install starting",
		zap.Int64("id", id), zap.String("game", g.ID), zap.String("source", g.Source),
		zap.Int("appid", appid), zap.String("steamUser", steamUser))
	result, err := a.inst.Install(ctx, toInstallerGame(g), s.Name, s.Port, steamUser, "")
	if err != nil {
		a.logger.Error("install failed", zap.Int64("id", id), zap.Error(err))
		_ = a.store.UpdateServerLastError(id, err.Error())
		_ = a.store.UpdateServerStatus(id, "install-error")
		return
	}
	a.logger.Info("install ok", zap.Int64("id", id),
		zap.String("dir", result.InstallDir), zap.String("start", result.StartCmd))
	_ = a.store.UpdateServerInstall(id, result.InstallDir, result.StartCmd)
	_ = a.store.UpdateServerStatus(id, "stopped")
}

func toInstallerGame(g Game) installer.Game {
	ig := installer.Game{
		ID:          g.ID,
		Name:        g.Name,
		Source:      g.Source,
		DefaultPort: g.DefaultPort,
	}
	if g.InstallRecipe != nil {
		ig.Recipe = &installer.Recipe{
			Method:     g.InstallRecipe.Method,
			URL:        g.InstallRecipe.URL,
			SteamAppID: g.InstallRecipe.SteamAppID,
			SteamArgs:  g.InstallRecipe.SteamArgs,
		}
		for _, f := range g.InstallRecipe.AdditionalURLs {
			ig.Recipe.AdditionalURLs = append(ig.Recipe.AdditionalURLs,
				installer.FileFetch{URL: f.URL, Dest: f.Dest})
		}
		for _, f := range g.InstallRecipe.PostInstallFiles {
			ig.Recipe.PostInstallFiles = append(ig.Recipe.PostInstallFiles,
				installer.FileWrite{Path: f.Path, Content: f.Content})
		}
	}
	if g.Start != nil {
		ig.Start = &installer.Start{
			Binary:    g.Start.Binary,
			Command:   g.Start.Command,
			Wrap:      g.Start.Wrap,
			ExtraLibs: g.Start.ExtraLibs,
			Args:      g.Start.Args,
		}
	}
	return ig
}
