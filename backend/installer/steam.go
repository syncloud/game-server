package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

type SteamInstaller struct {
	cmdPath string
	lib32   string
	lib64   string
}

func NewSteamInstaller(cmdPath, lib32, lib64 string) *SteamInstaller {
	return &SteamInstaller{cmdPath: cmdPath, lib32: lib32, lib64: lib64}
}

func (s *SteamInstaller) Install(ctx context.Context, g Game, installDir, user, pass string) (*Result, error) {
	if _, err := os.Stat(s.cmdPath); err != nil {
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
	cmd := exec.CommandContext(ctx, s.cmdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = installDir
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("steamcmd: %w", err)
	}
	return &Result{
		InstallDir: installDir,
		StartCmd:   s.startCmd(g, installDir),
	}, nil
}

func (s *SteamInstaller) startCmd(g Game, dir string) string {
	switch g.ID {
	case "cs2":
		bin := dir + "/game/bin/linuxsteamrt64/cs2"
		return s.wrapAmd64(bin, dir+"/game/bin/linuxsteamrt64", fmt.Sprintf("-dedicated +map de_dust2 +port %d", g.DefaultPort))
	case "tf2":
		return s.wrapI386(dir+"/srcds_linux", dir+":"+dir+"/bin", fmt.Sprintf("-game tf +map ctf_2fort +port %d", g.DefaultPort))
	case "gmod":
		return s.wrapI386(dir+"/srcds_linux", dir+":"+dir+"/bin", fmt.Sprintf("-game garrysmod +port %d", g.DefaultPort))
	case "valheim":
		return s.wrapAmd64(dir+"/valheim_server.x86_64", dir, fmt.Sprintf("-port %d -world Dedicated -password changeme", g.DefaultPort))
	case "zomboid":
		return fmt.Sprintf("cd %s && ./start-server.sh -port %d", dir, g.DefaultPort)
	case "hlds-cs":
		return s.wrapI386(dir+"/hlds_linux", dir+":"+dir+"/cstrike",
			fmt.Sprintf("-game cstrike -insecure +sv_lan 1 +map de_dust2 +port %d +maxplayers 8", g.DefaultPort))
	default:
		return fmt.Sprintf("echo 'no default startCmd for %s; configure manually'", g.ID)
	}
}

func (s *SteamInstaller) wrapAmd64(binary, extraPaths, args string) string {
	libs := s.lib64
	if extraPaths != "" {
		libs = libs + ":" + extraPaths
	}
	return fmt.Sprintf(
		"LD_LIBRARY_PATH=%s %s/ld-linux-x86-64.so.2 --library-path %s %s %s",
		libs, s.lib64, libs, binary, args)
}

func (s *SteamInstaller) wrapI386(binary, extraPaths, args string) string {
	libs := s.lib32
	if extraPaths != "" {
		libs = libs + ":" + extraPaths
	}
	return fmt.Sprintf(
		"LD_LIBRARY_PATH=%s %s/ld-linux.so.2 --library-path %s %s %s",
		libs, s.lib32, libs, binary, args)
}
