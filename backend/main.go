package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
)

const socketPath = "/var/snap/game-server/current/backend.sock"

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

type Server struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	GameID string `json:"gameId"`
	Status string `json:"status"`
	Port   int    `json:"port"`
}

var servers = []Server{}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	logger := log.New(os.Stdout, "backend: ", log.LstdFlags)

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
		writeJSON(w, http.StatusOK, servers)
	})

	logger.Printf("listening on %s", socketPath)
	if err := http.Serve(listener, mux); err != nil {
		logger.Fatalf("serve: %v", err)
	}
}
