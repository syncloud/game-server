package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/syncloud/game-server/backend/catalog"
	"github.com/syncloud/game-server/backend/db"
	"github.com/syncloud/game-server/backend/installer"
	"github.com/syncloud/game-server/backend/query"
	"github.com/syncloud/game-server/backend/runner"
	"github.com/syncloud/game-server/backend/server"
)

type Game = catalog.Game

const socketPath = "/var/snap/game-server/current/backend.sock"
const dbPath = "/var/snap/game-server/current/database.db"


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

	store, err := openStore(logger)
	if err != nil {
		logger.Fatalf("db: %v", err)
	}
	run := runner.New(logger)

	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		logger.Fatalf("listen: %v", err)
	}
	if err := os.Chmod(socketPath, 0666); err != nil {
		logger.Fatalf("chmod: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/games", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, catalog.All())
	})
	mux.HandleFunc("/api/v1/catalog/sources", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, catalog.Sources())
	})
	mux.HandleFunc("/api/v1/servers", func(w http.ResponseWriter, r *http.Request) {
		handleServers(w, r, store)
	})
	mux.HandleFunc("/api/v1/servers/", func(w http.ResponseWriter, r *http.Request) {
		handleServerByID(w, r, store, run)
	})

	logger.Printf("listening on %s", socketPath)
	if err := http.Serve(listener, mux); err != nil {
		logger.Fatalf("serve: %v", err)
	}
}

func openStore(logger *log.Logger) (*server.Store, error) {
	d, err := db.Open(dbPath)
	if err != nil {
		if _, statErr := os.Stat("/var/snap/game-server/current"); statErr != nil {
			logger.Printf("data dir missing, falling back to in-memory db: %v", statErr)
			d, err = db.Open(":memory:")
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return server.NewStore(d.DB), nil
}

type createRequest struct {
	Name     string `json:"name"`
	GameID   string `json:"gameId"`
	Port     int    `json:"port"`
	StartCmd string `json:"startCmd"`
}

func handleServers(w http.ResponseWriter, r *http.Request, store *server.Store) {
	switch r.Method {
	case http.MethodGet:
		list, err := store.List()
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
		s, err := store.Create(server.Server{
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

func handleServerByID(w http.ResponseWriter, r *http.Request, store *server.Store, run *runner.Runner) {
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
			handleServerAction(w, r, store, run, id, parts[1])
			return
		}
	}
	switch r.Method {
	case http.MethodGet:
		s, err := store.Get(id)
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
		if err := store.Delete(id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleServerAction(w http.ResponseWriter, r *http.Request, store *server.Store, run *runner.Runner, id int64, action string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s, err := store.Get(id)
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
		_ = store.UpdateStatus(id, "installing")
		go runInstall(log.Default(), store, id, *game)
		s.Status = "installing"
	case "start":
		if err := run.Start(id, s.StartCmd, s.InstallDir); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		_ = store.UpdateStatus(id, "running")
	case "stop":
		if err := run.Stop(id); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		_ = store.UpdateStatus(id, "stopped")
	case "restart":
		if err := run.Restart(id, s.StartCmd, s.InstallDir); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		_ = store.UpdateStatus(id, "running")
	default:
		writeError(w, http.StatusNotFound, "unknown action")
		return
	}
	s, _ = store.Get(id)
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

func handleQuery(w http.ResponseWriter, r *http.Request, store *server.Store, id int64) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s, err := store.Get(id)
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

func currentStatus(s *server.Server, run *runner.Runner) string {
	if s.Status == "installing" || s.Status == "install-error" {
		return s.Status
	}
	if run.Running(s.ID) {
		return "running"
	}
	return "stopped"
}

func runInstall(logger *log.Logger, store *server.Store, id int64, g Game) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	s, err := store.Get(id)
	if err != nil || s == nil {
		return
	}
	logger.Printf("install[%d] starting: game=%s source=%s appid=%d egg=%s", id, g.ID, g.Source, g.SteamAppID, g.EggURL)
	result, err := installer.Install(ctx, installer.Game{
		ID:          g.ID,
		Name:        g.Name,
		Source:      g.Source,
		SteamAppID:  g.SteamAppID,
		EggURL:      g.EggURL,
		DefaultPort: g.DefaultPort,
	}, s.Name, s.Port, "", "")
	if err != nil {
		logger.Printf("install[%d] FAILED: %v", id, err)
		_ = store.UpdateLastError(id, err.Error())
		_ = store.UpdateStatus(id, "install-error")
		return
	}
	logger.Printf("install[%d] OK: dir=%s start=%q", id, result.InstallDir, result.StartCmd)
	_ = store.UpdateInstall(id, result.InstallDir, result.StartCmd)
	_ = store.UpdateStatus(id, "stopped")
}
