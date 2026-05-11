package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/syncloud/game-server/backend/db"
	"github.com/syncloud/game-server/backend/server"
)

const socketPath = "/var/snap/game-server/current/backend.sock"
const dbPath = "/var/snap/game-server/current/database.db"

type Game struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Source      string   `json:"source"`
	SteamAppID  int      `json:"steamAppId,omitempty"`
	EggURL      string   `json:"eggUrl,omitempty"`
	Summary     string   `json:"summary"`
	DefaultPort int      `json:"defaultPort"`
	Protocols   []string `json:"protocols"`
}

var catalog = []Game{
	{ID: "cs2", Name: "Counter-Strike 2", Source: "steam", SteamAppID: 730, Summary: "Valve's tactical shooter dedicated server.", DefaultPort: 27015, Protocols: []string{"udp"}},
	{ID: "tf2", Name: "Team Fortress 2", Source: "steam", SteamAppID: 232250, Summary: "Class-based team shooter.", DefaultPort: 27015, Protocols: []string{"udp"}},
	{ID: "gmod", Name: "Garry's Mod", Source: "steam", SteamAppID: 4020, Summary: "Sandbox modification of Source.", DefaultPort: 27015, Protocols: []string{"udp"}},
	{ID: "valheim", Name: "Valheim", Source: "steam", SteamAppID: 896660, Summary: "Viking survival co-op.", DefaultPort: 2456, Protocols: []string{"udp"}},
	{ID: "rust", Name: "Rust", Source: "steam", SteamAppID: 258550, Summary: "Multiplayer survival.", DefaultPort: 28015, Protocols: []string{"udp"}},
	{ID: "zomboid", Name: "Project Zomboid", Source: "steam", SteamAppID: 380870, Summary: "Isometric zombie survival sandbox.", DefaultPort: 16261, Protocols: []string{"udp"}},
	{ID: "ark", Name: "ARK: Survival Evolved", Source: "steam", SteamAppID: 376030, Summary: "Dinosaur survival multiplayer.", DefaultPort: 7777, Protocols: []string{"udp"}},
	{ID: "teeworlds", Name: "Teeworlds", Source: "egg", EggURL: "https://raw.githubusercontent.com/parkervcp/eggs/master/game_eggs/teeworlds/egg-teeworlds.json", Summary: "Tiny 2D competitive shooter. Smallest server, used as our CI fixture.", DefaultPort: 8303, Protocols: []string{"udp"}},
	{ID: "minetest", Name: "Minetest", Source: "egg", EggURL: "https://raw.githubusercontent.com/parkervcp/eggs/master/game_eggs/minetest/egg-minetest.json", Summary: "Open-source voxel sandbox.", DefaultPort: 30000, Protocols: []string{"udp"}},
	{ID: "minecraft-java", Name: "Minecraft (Java)", Source: "egg", EggURL: "https://raw.githubusercontent.com/parkervcp/eggs/master/minecraft/java/vanilla/egg-vanilla-minecraft.json", Summary: "Vanilla Minecraft Java edition server.", DefaultPort: 25565, Protocols: []string{"tcp"}},
	{ID: "terraria", Name: "Terraria (TShock)", Source: "egg", EggURL: "https://raw.githubusercontent.com/parkervcp/eggs/master/game_eggs/terraria/tshock/egg-t-shock.json", Summary: "2D sandbox with TShock server.", DefaultPort: 7777, Protocols: []string{"tcp"}},
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
	for i := range catalog {
		if catalog[i].ID == id {
			return &catalog[i]
		}
	}
	return nil
}

func main() {
	logger := log.New(os.Stdout, "backend: ", log.LstdFlags)

	store, err := openStore(logger)
	if err != nil {
		logger.Fatalf("db: %v", err)
	}

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
		writeJSON(w, http.StatusOK, catalog)
	})
	mux.HandleFunc("/api/v1/servers", func(w http.ResponseWriter, r *http.Request) {
		handleServers(w, r, store)
	})
	mux.HandleFunc("/api/v1/servers/", func(w http.ResponseWriter, r *http.Request) {
		handleServerByID(w, r, store)
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
	Name   string `json:"name"`
	GameID string `json:"gameId"`
	Port   int    `json:"port"`
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
		s, err := store.Create(server.Server{Name: req.Name, GameID: req.GameID, Port: req.Port})
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

func handleServerByID(w http.ResponseWriter, r *http.Request, store *server.Store) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
	parts := strings.SplitN(rest, "/", 2)
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
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
		writeJSON(w, http.StatusOK, s)
	case http.MethodDelete:
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
