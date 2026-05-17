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

type Egg struct {
	Name    string `json:"name"`
	Startup string `json:"startup"`
	Scripts struct {
		Installation struct {
			Script     string `json:"script"`
			Container  string `json:"container"`
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

type EggInstaller struct {
	jreBin     string
	httpClient *http.Client
}

func NewEggInstaller(jreBin string) *EggInstaller {
	return &EggInstaller{
		jreBin:     jreBin,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *EggInstaller) Install(ctx context.Context, g Game, installDir string) (*Result, error) {
	if g.ID == "teeworlds" {
		return e.installTeeworldsNative(ctx, g, installDir)
	}
	if g.EggURL == "" {
		return nil, fmt.Errorf("egg url empty")
	}
	egg, err := e.fetchEgg(ctx, g.EggURL)
	if err != nil {
		return nil, err
	}
	scriptPath := filepath.Join(installDir, ".install.sh")
	if err := os.WriteFile(scriptPath, []byte(prependShebang(egg.Scripts.Installation.Script, egg.Scripts.Installation.Entrypoint)), 0755); err != nil {
		return nil, fmt.Errorf("write install: %w", err)
	}
	env := append(os.Environ(), eggEnv(egg, g, installDir)...)
	env = append(env, "PATH="+e.jreBin+":"+os.Getenv("PATH"))
	cmd := exec.CommandContext(ctx, scriptPath)
	cmd.Dir = installDir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("egg install: %w", err)
	}
	startCmd := e.renderStartup(egg, g, installDir)
	return &Result{InstallDir: installDir, StartCmd: startCmd}, nil
}

func (e *EggInstaller) fetchEgg(ctx context.Context, url string) (*Egg, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := e.httpClient.Do(req)
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

func (e *EggInstaller) renderStartup(egg *Egg, g Game, installDir string) string {
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
	return fmt.Sprintf("cd %s && export PATH=%s:$PATH && %s", installDir, e.jreBin, startup)
}
