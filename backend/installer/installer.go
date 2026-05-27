package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

const (
	SteamCMDPath   = "/snap/games/current/bin/steamcmd.sh"
	SteamLib32     = "/snap/games/current/steamcmd/lib32"
	SteamLib64     = "/snap/games/current/steamcmd/lib64"
	ServersBaseDir = "/data/games/servers"
)

type Game struct {
	ID          string
	Name        string
	Source      string
	DefaultPort int
	Recipe      *Recipe
	Start       *Start
}

type Recipe struct {
	Method     string
	URL        string
	SteamAppID int
	SteamArgs  []string
}

type Start struct {
	Binary    string
	Wrap      string
	ExtraLibs []string
	Args      string
}

type Result struct {
	InstallDir string
	StartCmd   string
}

type Installer struct {
	serversDir string
	recipe     *RecipeInstaller
}

func New(serversDir string, recipe *RecipeInstaller) *Installer {
	return &Installer{serversDir: serversDir, recipe: recipe}
}

func (i *Installer) Install(ctx context.Context, g Game, name string, port int, steamUser, steamPass string) (*Result, error) {
	installDir := filepath.Join(i.serversDir, name)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	if port > 0 {
		g.DefaultPort = port
	}
	return i.recipe.Install(ctx, g, installDir, steamUser, steamPass)
}
