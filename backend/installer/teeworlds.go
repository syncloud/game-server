package installer

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ulikunitz/xz"
)

const (
	teeworldsVersion  = "0.7.5"
	teeworldsAssetURL = "https://github.com/teeworlds/teeworlds/releases/download/" + teeworldsVersion + "/teeworlds-" + teeworldsVersion + "-linux_x86_64.tar.gz"
)

func installTeeworldsNative(ctx context.Context, g Game, installDir string) (*Result, error) {
	tarballPath := filepath.Join(installDir, filepath.Base(teeworldsAssetURL))
	if err := download(ctx, teeworldsAssetURL, tarballPath); err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer os.Remove(tarballPath)

	if err := extractTar(tarballPath, installDir); err != nil {
		return nil, fmt.Errorf("extract: %w", err)
	}

	srvPath, err := findFile(installDir, "teeworlds_srv")
	if err != nil {
		return nil, fmt.Errorf("locate teeworlds_srv: %w", err)
	}
	if err := os.Chmod(srvPath, 0755); err != nil {
		return nil, fmt.Errorf("chmod: %w", err)
	}

	startCmd := fmt.Sprintf("cd %s && %s \"sv_port %d\"", filepath.Dir(srvPath), srvPath, g.DefaultPort)
	return &Result{InstallDir: installDir, StartCmd: startCmd}, nil
}

func download(ctx context.Context, url, dst string) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
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

func extractTar(tarballPath, dst string) error {
	f, err := os.Open(tarballPath)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader
	switch {
	case strings.HasSuffix(tarballPath, ".tar.xz") || strings.HasSuffix(tarballPath, ".txz"):
		xzr, err := xz.NewReader(f)
		if err != nil {
			return fmt.Errorf("xz: %w", err)
		}
		r = xzr
	case strings.HasSuffix(tarballPath, ".tar.gz") || strings.HasSuffix(tarballPath, ".tgz"):
		gzr, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("gzip: %w", err)
		}
		defer gzr.Close()
		r = gzr
	default:
		return fmt.Errorf("unknown archive extension: %s", filepath.Base(tarballPath))
	}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dst, hdr.Name)
		if !strings.HasPrefix(target, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("tar entry escapes destination: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
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
