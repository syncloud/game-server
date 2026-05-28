package installer

import (
	"context"
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

const archiversDir = "/snap/games/current/archivers"

type RecipeInstaller struct {
	steamCMDPath string
	steamLib32   string
	steamLib64   string
	httpClient   *http.Client
}

func NewRecipeInstaller(steamCMDPath, lib32, lib64 string) *RecipeInstaller {
	return &RecipeInstaller{
		steamCMDPath: steamCMDPath,
		steamLib32:   lib32,
		steamLib64:   lib64,
		httpClient:   &http.Client{Timeout: 30 * time.Minute},
	}
}

func (r *RecipeInstaller) Install(ctx context.Context, g Game, installDir, steamUser, steamPass string) (*Result, error) {
	if g.Recipe == nil {
		return nil, fmt.Errorf("no install recipe for %s", g.ID)
	}
	switch g.Recipe.Method {
	case "steam":
		return r.installSteam(ctx, g, installDir, steamUser, steamPass)
	case "downloadExtract":
		return r.installDownloadExtract(ctx, g, installDir)
	default:
		return nil, fmt.Errorf("unknown install method %q", g.Recipe.Method)
	}
}

func (r *RecipeInstaller) installSteam(ctx context.Context, g Game, installDir, user, pass string) (*Result, error) {
	if _, err := os.Stat(r.steamCMDPath); err != nil {
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
	if len(g.Recipe.SteamArgs) > 0 {
		args = append(args, g.Recipe.SteamArgs...)
	} else {
		args = append(args, "+app_update", strconv.Itoa(g.Recipe.SteamAppID), "validate")
	}
	args = append(args, "+quit")
	cmd := exec.CommandContext(ctx, r.steamCMDPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = installDir
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("steamcmd: %w", err)
	}
	return &Result{InstallDir: installDir, StartCmd: r.renderStart(g, installDir)}, nil
}

func (r *RecipeInstaller) installDownloadExtract(ctx context.Context, g Game, installDir string) (*Result, error) {
	url := g.Recipe.URL
	if url == "" {
		return nil, fmt.Errorf("recipe url empty")
	}
	tarballPath := filepath.Join(installDir, filepath.Base(url))
	if err := r.download(ctx, url, tarballPath); err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer os.Remove(tarballPath)
	if err := extractArchive(tarballPath, installDir); err != nil {
		return nil, fmt.Errorf("extract: %w", err)
	}
	if g.Start != nil && g.Start.Binary != "" {
		binPath, err := findFile(installDir, filepath.Base(g.Start.Binary))
		if err == nil {
			_ = os.Chmod(binPath, 0755)
			if rel, relErr := filepath.Rel(installDir, binPath); relErr == nil {
				g.Start.Binary = rel
			}
		}
	}
	return &Result{InstallDir: installDir, StartCmd: r.renderStart(g, installDir)}, nil
}

func (r *RecipeInstaller) renderStart(g Game, installDir string) string {
	if g.Start == nil {
		return fmt.Sprintf("echo 'no start recipe for %s; configure manually'", g.ID)
	}
	args := strings.ReplaceAll(g.Start.Args, "{{port}}", strconv.Itoa(g.DefaultPort))
	bin := filepath.Join(installDir, g.Start.Binary)
	switch g.Start.Wrap {
	case "i386":
		return r.wrap(r.steamLib32, "ld-linux.so.2", installDir, g.Start.ExtraLibs, bin, args)
	case "amd64":
		return r.wrap(r.steamLib64, "ld-linux-x86-64.so.2", installDir, g.Start.ExtraLibs, bin, args)
	default:
		return fmt.Sprintf("cd %s && ./%s %s", installDir, g.Start.Binary, args)
	}
}

func (r *RecipeInstaller) wrap(libBase, loader, installDir string, extras []string, bin, args string) string {
	paths := []string{libBase}
	for _, p := range extras {
		if p == "." || p == "" {
			paths = append(paths, installDir)
		} else {
			paths = append(paths, filepath.Join(installDir, p))
		}
	}
	libPath := strings.Join(paths, ":")
	return fmt.Sprintf(
		"LD_LIBRARY_PATH=%s %s/%s --library-path %s %s %s",
		libPath, libBase, loader, libPath, bin, args)
}

func (r *RecipeInstaller) download(ctx context.Context, url, dst string) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; syncloud-games)")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func extractArchive(archivePath, dst string) error {
	var cmd *exec.Cmd
	switch {
	case strings.HasSuffix(archivePath, ".zip"):
		cmd = exec.Command(archiversDir+"/bin/unzip", "-qq", "-o", archivePath, "-d", dst)
	case strings.HasSuffix(archivePath, ".tar.xz"), strings.HasSuffix(archivePath, ".txz"),
		strings.HasSuffix(archivePath, ".tar.gz"), strings.HasSuffix(archivePath, ".tgz"),
		strings.HasSuffix(archivePath, ".tar.bz2"), strings.HasSuffix(archivePath, ".tbz2"),
		strings.HasSuffix(archivePath, ".tar"):
		cmd = exec.Command(archiversDir+"/bin/tar", "-xf", archivePath, "-C", dst)
	default:
		return fmt.Errorf("unsupported archive: %s", filepath.Base(archivePath))
	}
	cmd.Env = append(os.Environ(),
		"LD_LIBRARY_PATH="+archiversDir+"/lib",
		"PATH="+archiversDir+"/bin:"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %v: %s",
			filepath.Base(cmd.Path), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func findFile(root, name string) (string, error) {
	var found string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == name {
			found = p
			return io.EOF
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("%s not found in %s", name, root)
	}
	return found, nil
}

