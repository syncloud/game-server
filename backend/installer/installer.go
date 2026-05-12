package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	SteamCMDPath   = "/snap/game-server/current/bin/steamcmd.sh"
	SteamLib32     = "/snap/game-server/current/steamcmd/lib32"
	ServersBaseDir = "/var/snap/game-server/current/servers"
)

type Game struct {
	ID          string
	Name        string
	Source      string
	SteamAppID  int
	EggURL      string
	DefaultPort int
}

type Result struct {
	InstallDir string
	StartCmd   string
}

type Egg struct {
	Name    string `json:"name"`
	Startup string `json:"startup"`
	Scripts struct {
		Installation struct {
			Script    string `json:"script"`
			Container string `json:"container"`
			Entrypoint string `json:"entrypoint"`
		} `json:"installation"`
	} `json:"scripts"`
	Variables []EggVariable `json:"variables"`
}

type EggVariable struct {
	Name         string `json:"name"`
	EnvVariable  string `json:"env_variable"`
	DefaultValue string `json:"default_value"`
}

func Install(ctx context.Context, g Game, name string, port int, steamUser, steamPass string) (*Result, error) {
	installDir := filepath.Join(ServersBaseDir, name)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	if port > 0 {
		g.DefaultPort = port
	}
	switch g.Source {
	case "steam":
		return installSteam(ctx, g, installDir, steamUser, steamPass)
	case "egg":
		return installEgg(ctx, g, installDir)
	default:
		return nil, fmt.Errorf("unknown source %q", g.Source)
	}
}

func installSteam(ctx context.Context, g Game, installDir, user, pass string) (*Result, error) {
	if _, err := os.Stat(SteamCMDPath); err != nil {
		return nil, fmt.Errorf("steamcmd not bundled: %w", err)
	}
	login := "anonymous"
	if user != "" {
		login = user
		if pass != "" {
			login += " " + pass
		}
	}
	args := []string{
		"+@sSteamCmdForcePlatformType", "linux",
		"+force_install_dir", installDir,
		"+login", login,
		"+app_update", strconv.Itoa(g.SteamAppID), "validate",
		"+quit",
	}
	cmd := exec.CommandContext(ctx, SteamCMDPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("steamcmd: %w", err)
	}
	return &Result{
		InstallDir: installDir,
		StartCmd:   steamStartCmd(g, installDir),
	}, nil
}

func steamStartCmd(g Game, dir string) string {
	switch g.ID {
	case "cs2":
		return fmt.Sprintf("%s/game/bin/linuxsteamrt64/cs2 -dedicated +map de_dust2", dir)
	case "tf2":
		return fmt.Sprintf("%s/srcds_run -game tf -port %d", dir, g.DefaultPort)
	case "gmod":
		return fmt.Sprintf("%s/srcds_run -game garrysmod -port %d", dir, g.DefaultPort)
	case "valheim":
		return fmt.Sprintf("%s/valheim_server.x86_64 -port %d -world \"Dedicated\" -password \"changeme\"", dir, g.DefaultPort)
	case "zomboid":
		return fmt.Sprintf("%s/start-server.sh", dir)
	case "hlds-cs":
		// HLDS uses the same bundled 32-bit loader as steamcmd
		return fmt.Sprintf(
			"LD_LIBRARY_PATH=%s:%s:%s/cstrike %s/ld-linux.so.2 --library-path %s:%s:%s/cstrike %s/hlds_linux -game cstrike +map de_dust2 +port %d",
			SteamLib32, dir, dir,
			SteamLib32,
			SteamLib32, dir, dir,
			dir, g.DefaultPort)
	default:
		return fmt.Sprintf("echo 'no default startCmd for %s; configure manually'", g.ID)
	}
}

func installEgg(ctx context.Context, g Game, installDir string) (*Result, error) {
	if g.ID == "teeworlds" {
		return installTeeworldsNative(ctx, g, installDir)
	}
	if g.EggURL == "" {
		return nil, fmt.Errorf("egg url empty")
	}
	egg, err := fetchEgg(ctx, g.EggURL)
	if err != nil {
		return nil, err
	}
	scriptPath := filepath.Join(installDir, ".install.sh")
	if err := os.WriteFile(scriptPath, []byte(prependShebang(egg.Scripts.Installation.Script, egg.Scripts.Installation.Entrypoint)), 0755); err != nil {
		return nil, fmt.Errorf("write install: %w", err)
	}
	env := append(os.Environ(), eggEnv(egg, g, installDir)...)
	cmd := exec.CommandContext(ctx, scriptPath)
	cmd.Dir = installDir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("egg install: %w", err)
	}
	startCmd := renderStartup(egg, g, installDir)
	return &Result{InstallDir: installDir, StartCmd: startCmd}, nil
}

func fetchEgg(ctx context.Context, url string) (*Egg, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch egg: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("egg http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var egg Egg
	if err := json.Unmarshal(body, &egg); err != nil {
		return nil, fmt.Errorf("egg json: %w", err)
	}
	return &egg, nil
}

func prependShebang(script, entrypoint string) string {
	if entrypoint == "" {
		entrypoint = "bash"
	}
	if strings.HasPrefix(script, "#!") {
		return script
	}
	return "#!/usr/bin/env " + entrypoint + "\n" + script
}

func eggEnv(egg *Egg, g Game, installDir string) []string {
	env := []string{
		"SERVER_DIR=" + installDir,
		"SERVER_PORT=" + strconv.Itoa(g.DefaultPort),
	}
	for _, v := range egg.Variables {
		env = append(env, v.EnvVariable+"="+v.DefaultValue)
	}
	return env
}

func renderStartup(egg *Egg, g Game, installDir string) string {
	startup := egg.Startup
	startup = strings.ReplaceAll(startup, "{{SERVER_PORT}}", strconv.Itoa(g.DefaultPort))
	startup = strings.ReplaceAll(startup, "{{SERVER_MEMORY}}", "1024")
	for _, v := range egg.Variables {
		placeholder := "{{" + v.EnvVariable + "}}"
		startup = strings.ReplaceAll(startup, placeholder, v.DefaultValue)
	}
	if startup == "" {
		return fmt.Sprintf("echo 'no startup defined for %s'", g.ID)
	}
	return fmt.Sprintf("cd %s && %s", installDir, startup)
}
