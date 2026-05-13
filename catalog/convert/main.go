// catalog converter: walks parkervcp/eggs and pelican-eggs/games, parses
// each egg-*.json, applies filter rules to assign a tier, dedupes, writes
// a single catalog.json that the snap embeds at build time.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Egg struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Image       string `json:"image"`
	Startup     string `json:"startup"`
	Scripts     struct {
		Installation struct {
			Script     string `json:"script"`
			Container  string `json:"container"`
			Entrypoint string `json:"entrypoint"`
		} `json:"installation"`
	} `json:"scripts"`
	Variables []struct {
		Name         string `json:"name"`
		EnvVariable  string `json:"env_variable"`
		DefaultValue string `json:"default_value"`
	} `json:"variables"`
}

type CatalogGame struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Source      string        `json:"source"`
	UpstreamRef string        `json:"upstreamRef"`
	Summary     string        `json:"summary"`
	DefaultPort int           `json:"defaultPort"`
	Protocols   []string      `json:"protocols"`
	Tier        string        `json:"tier"`
	TierReason  string        `json:"tierReason,omitempty"`
	EggURL      string        `json:"eggUrl,omitempty"`
	EggInline   *EggInline    `json:"egg,omitempty"`
	SteamAppID  int           `json:"steamAppId,omitempty"`
	Variables   []EnvVariable `json:"variables,omitempty"`
}

type EggInline struct {
	InstallScript     string `json:"installScript"`
	InstallEntrypoint string `json:"installEntrypoint"`
	Startup           string `json:"startup"`
}

type EnvVariable struct {
	Env     string `json:"env"`
	Default string `json:"default"`
}

type Catalog struct {
	Sources map[string]string `json:"sources"`
	Games   []CatalogGame     `json:"games"`
}

var (
	dockerContainerRE = regexp.MustCompile(`(?i)ghcr\.io/(parkervcp|pelican-eggs)/yolks`)
	steamAppRE        = regexp.MustCompile(`\+app_update\s+(\d+)`)
	portRE            = regexp.MustCompile(`\b([0-9]{4,5})\b`)
	slugRE            = regexp.MustCompile(`[^a-z0-9]+`)
)

var (
	parkervcpRoot string
	pelicanRoot   string
	parkervcpRef  string
	pelicanRef    string
)

func main() {
	parkervcp := flag.String("parkervcp", "", "path to parkervcp game_eggs/")
	pelican := flag.String("pelican", "", "path to pelican-eggs games/")
	parkervcpVer := flag.String("parkervcp-version", "", "")
	pelicanVer := flag.String("pelican-version", "", "")
	out := flag.String("out", "", "output catalog.json path")
	flag.Parse()
	parkervcpRoot = *parkervcp
	pelicanRoot = *pelican
	parkervcpRef = *parkervcpVer
	pelicanRef = *pelicanVer

	cat := Catalog{
		Sources: map[string]string{
			"parkervcp/eggs":      *parkervcpVer,
			"pelican-eggs/games":  *pelicanVer,
		},
	}

	games := map[string]CatalogGame{}
	if *parkervcp != "" {
		walkAndIngest(*parkervcp, "parkervcp", games)
	}
	if *pelican != "" {
		walkAndIngest(*pelican, "pelican", games)
	}

	for _, g := range games {
		cat.Games = append(cat.Games, g)
	}
	sort.Slice(cat.Games, func(i, j int) bool { return cat.Games[i].ID < cat.Games[j].ID })

	if *out == "" {
		log.Fatal("--out required")
	}
	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cat); err != nil {
		log.Fatal(err)
	}
	f.Close()
	fmt.Fprintf(os.Stderr, "wrote %d games to %s\n", len(cat.Games), *out)
}

func walkAndIngest(root, sourceLabel string, out map[string]CatalogGame) {
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if !strings.HasPrefix(base, "egg-") || !strings.HasSuffix(base, ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var egg Egg
		if json.Unmarshal(data, &egg) != nil {
			return nil
		}
		g := convertEgg(egg, sourceLabel, path, root)
		if g == nil {
			return nil
		}
		// Prefer earlier-seen (parkervcp before pelican on iteration order, but
		// since we may want pelican as the more current source, take the later
		// one). Easier: prefer pelican over parkervcp on conflict.
		if existing, ok := out[g.ID]; ok {
			if existing.Source == "parkervcp" && g.Source == "pelican" {
				out[g.ID] = *g
			}
			return nil
		}
		out[g.ID] = *g
		return nil
	})
}

func convertEgg(egg Egg, source, path, root string) *CatalogGame {
	name := strings.TrimSpace(egg.Name)
	if name == "" {
		return nil
	}
	id := slugRE.ReplaceAllString(strings.ToLower(name), "-")
	id = strings.Trim(id, "-")
	if id == "" {
		return nil
	}

	rel := relPath(path, root)
	g := CatalogGame{
		ID:          id,
		Name:        name,
		Source:      source,
		UpstreamRef: rel,
		EggURL:      eggURL(source, rel),
		Summary:     truncate(egg.Description, 200),
		Protocols:   detectProtocols(egg.Startup),
		DefaultPort: detectPort(egg),
		EggInline: &EggInline{
			InstallScript:     egg.Scripts.Installation.Script,
			InstallEntrypoint: defaultStr(egg.Scripts.Installation.Entrypoint, "bash"),
			Startup:           egg.Startup,
		},
	}
	if m := steamAppRE.FindStringSubmatch(egg.Scripts.Installation.Script); len(m) == 2 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			g.SteamAppID = n
		}
	}
	for _, v := range egg.Variables {
		g.Variables = append(g.Variables, EnvVariable{Env: v.EnvVariable, Default: v.DefaultValue})
	}

	g.Tier, g.TierReason = classify(egg, g)
	return &g
}

// classify returns the tier (supported|compatible|experimental) and an
// optional reason for downgrades.
func classify(egg Egg, g CatalogGame) (string, string) {
	ep := strings.ToLower(strings.TrimSpace(egg.Scripts.Installation.Entrypoint))
	if ep != "bash" && ep != "sh" && ep != "" {
		return "experimental", "install entrypoint is " + ep + " (not bash/sh)"
	}
	container := strings.ToLower(egg.Scripts.Installation.Container)
	// Generic base images we can run install scripts against.
	if !strings.Contains(container, "debian") &&
		!strings.Contains(container, "ubuntu") &&
		!strings.Contains(container, "alpine") {
		return "experimental", "install container is " + egg.Scripts.Installation.Container + " (need debian/ubuntu/alpine)"
	}
	// If the runtime image is a Docker-only "yolk", we can't reliably reproduce
	// that environment in the snap — best-effort install at most.
	if dockerContainerRE.MatchString(egg.Image) && egg.Image != "" {
		return "compatible", "runtime image is a Pterodactyl/Pelican yolk; install may need adaptation"
	}
	// Install script that apt-get installs build deps tends to need root + apt
	// access we don't have at install time.
	if strings.Contains(egg.Scripts.Installation.Script, "apt update") ||
		strings.Contains(egg.Scripts.Installation.Script, "apt-get update") {
		return "compatible", "install script runs apt update (won't have apt at runtime)"
	}
	return "compatible", ""
}

func detectProtocols(startup string) []string {
	s := strings.ToLower(startup)
	if strings.Contains(s, "udp") {
		return []string{"udp"}
	}
	if strings.Contains(s, "tcp") {
		return []string{"tcp"}
	}
	return []string{"udp"}
}

func detectPort(egg Egg) int {
	for _, v := range egg.Variables {
		env := strings.ToUpper(v.EnvVariable)
		if env == "SERVER_PORT" || env == "PORT" || strings.HasSuffix(env, "_PORT") {
			if n, err := strconv.Atoi(strings.TrimSpace(v.DefaultValue)); err == nil && n > 0 {
				return n
			}
		}
	}
	if m := portRE.FindStringSubmatch(egg.Startup); len(m) == 2 {
		if n, err := strconv.Atoi(m[1]); err == nil && n >= 1024 && n <= 65535 {
			return n
		}
	}
	return 0
}

func eggURL(source, rel string) string {
	switch source {
	case "parkervcp":
		return fmt.Sprintf("https://raw.githubusercontent.com/parkervcp/eggs/%s/game_eggs/%s", parkervcpRef, rel)
	case "pelican":
		return fmt.Sprintf("https://raw.githubusercontent.com/pelican-eggs/games/%s/%s", pelicanRef, rel)
	}
	return ""
}

func relPath(p, root string) string {
	if r, err := filepath.Rel(root, p); err == nil {
		return r
	}
	return p
}

func defaultStr(s, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
