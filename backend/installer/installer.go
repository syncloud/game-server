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
	JREBinDir      = "/snap/games/current/jre/bin"
	ServersBaseDir = "/data/games/servers"
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

type Installer struct {
	serversDir string
	steam      *SteamInstaller
	egg        *EggInstaller
}

func New(serversDir string, steam *SteamInstaller, egg *EggInstaller) *Installer {
	return &Installer{serversDir: serversDir, steam: steam, egg: egg}
}

func (i *Installer) Install(ctx context.Context, g Game, name string, port int, steamUser, steamPass string) (*Result, error) {
	installDir := filepath.Join(i.serversDir, name)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	if port > 0 {
		g.DefaultPort = port
	}
	switch g.Source {
	case "steam", "linuxgsm":
		return i.steam.Install(ctx, g, installDir, steamUser, steamPass)
	case "egg", "pelican", "parkervcp":
		return i.egg.Install(ctx, g, installDir)
	default:
		return nil, fmt.Errorf("unknown source %q", g.Source)
	}
}
