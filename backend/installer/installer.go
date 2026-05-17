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
	SteamCMDPath = "/snap/games/current/bin/steamcmd.sh"
	SteamLib32   = "/snap/games/current/steamcmd/lib32"
	SteamLib64   = "/snap/games/current/steamcmd/lib64"
	JREBinDir    = "/snap/games/current/jre/bin"
	ServersBaseDir = "/data/games/servers"
)

func wrapAmd64(binary string, extraPaths string, args string) string {
	libs := SteamLib64
	if extraPaths != "" {
		libs = libs + ":" + extraPaths
	}
	return fmt.Sprintf(
		"LD_LIBRARY_PATH=%s %s/ld-linux-x86-64.so.2 --library-path %s %s %s",
		libs, SteamLib64, libs, binary, args)
}

func wrapI386(binary string, extraPaths string, args string) string {
	libs := SteamLib32
	if extraPaths != "" {
		libs = libs + ":" + extraPaths
	}
	return fmt.Sprintf(
		"LD_LIBRARY_PATH=%s %s/ld-linux.so.2 --library-path %s %s %s",
		libs, SteamLib32, libs, binary, args)
}

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
	case "steam", "linuxgsm":
		return installSteam(ctx, g, installDir, steamUser, steamPass)
	case "egg", "pelican", "parkervcp":
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
	}
	if g.ID == "hlds-cs" {
		args = append(args, "+app_set_config", "90", "mod", "cstrike")
		args = append(args, "+app_update", "90", "-beta", "steam_legacy", "validate")
	} else {
		args = append(args, "+app_update", strconv.Itoa(g.SteamAppID), "validate")
	}
	args = append(args, "+quit")
	cmd := exec.CommandContext(ctx, SteamCMDPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = installDir
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
		bin := dir + "/game/bin/linuxsteamrt64/cs2"
		return wrapAmd64(bin, dir+"/game/bin/linuxsteamrt64", fmt.Sprintf("-dedicated +map de_dust2 +port %d", g.DefaultPort))
	case "tf2":
		return wrapI386(dir+"/srcds_linux", dir+":"+dir+"/bin", fmt.Sprintf("-game tf +map ctf_2fort +port %d", g.DefaultPort))
	case "gmod":
		return wrapI386(dir+"/srcds_linux", dir+":"+dir+"/bin", fmt.Sprintf("-game garrysmod +port %d", g.DefaultPort))
	case "valheim":
		return wrapAmd64(dir+"/valheim_server.x86_64", dir, fmt.Sprintf("-port %d -world Dedicated -password changeme", g.DefaultPort))
	case "zomboid":
		return fmt.Sprintf("cd %s && ./start-server.sh -port %d", dir, g.DefaultPort)
	case "hlds-cs":
		return wrapI386(dir+"/hlds_linux", dir+":"+dir+"/cstrike",
			fmt.Sprintf("-game cstrike -insecure +sv_lan 1 +map de_dust2 +port %d +maxplayers 8", g.DefaultPort))
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
	env = append(env, "PATH="+JREBinDir+":"+os.Getenv("PATH"))
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
	return fmt.Sprintf("cd %s && export PATH=%s:$PATH && %s", installDir, JREBinDir, startup)
}
