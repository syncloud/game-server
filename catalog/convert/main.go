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
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Source        string         `json:"source"`
	UpstreamRef   string         `json:"upstreamRef,omitempty"`
	Summary       string         `json:"summary"`
	DefaultPort   int            `json:"defaultPort"`
	Protocols     []string       `json:"protocols"`
	Tier          string         `json:"tier"`
	TierReason    string         `json:"tierReason,omitempty"`
	SteamAppID    int            `json:"steamAppId,omitempty"`
	InstallRecipe *InstallRecipe `json:"installRecipe,omitempty"`
	Start         *StartRecipe   `json:"start,omitempty"`
}

type InstallRecipe struct {
	Method     string   `json:"method"`
	URL        string   `json:"url,omitempty"`
	SteamAppID int      `json:"steamAppId,omitempty"`
	SteamArgs  []string `json:"steamArgs,omitempty"`
}

type StartRecipe struct {
	Binary    string   `json:"binary"`
	Wrap      string   `json:"wrap,omitempty"`
	ExtraLibs []string `json:"extraLibs,omitempty"`
	Args      string   `json:"args,omitempty"`
}

type Catalog struct {
	Sources map[string]string `json:"sources"`
	Games   []CatalogGame     `json:"games"`
}

var (
	steamAppRE   = regexp.MustCompile(`\+app_update\s+(\d+)`)
	portRE       = regexp.MustCompile(`\b([0-9]{4,5})\b`)
	slugRE       = regexp.MustCompile(`[^a-z0-9]+`)
	literalURLRE = regexp.MustCompile(`https?://[^\s"'\\)]+`)
	archiveExtRE = regexp.MustCompile(`\.(tar\.gz|tgz|tar\.xz|tar\.bz2|tar|zip)(?:[?#]|$)`)
)

func main() {
	parkervcp := flag.String("parkervcp", "", "path to parkervcp game_eggs/")
	pelican := flag.String("pelican", "", "path to pelican-eggs games/")
	linuxgsm := flag.String("linuxgsm", "", "path to LinuxGSM checkout root")
	overridesPath := flag.String("overrides", "", "path to overrides.json (hand-curated)")
	parkervcpVer := flag.String("parkervcp-version", "", "")
	pelicanVer := flag.String("pelican-version", "", "")
	linuxgsmVer := flag.String("linuxgsm-version", "", "")
	out := flag.String("out", "", "output catalog.json path")
	flag.Parse()

	overrides := loadOverrides(*overridesPath)

	cat := Catalog{
		Sources: map[string]string{
			"parkervcp/eggs":     *parkervcpVer,
			"pelican-eggs/games": *pelicanVer,
			"linuxgsm":           *linuxgsmVer,
		},
	}

	games := map[string]CatalogGame{}
	if *parkervcp != "" {
		walkAndIngest(*parkervcp, "parkervcp", games)
	}
	if *pelican != "" {
		walkAndIngest(*pelican, "pelican", games)
	}
	if *linuxgsm != "" {
		walkLinuxGSM(*linuxgsm, games)
	}

	for id, ov := range overrides {
		base, ok := games[id]
		if !ok {
			base = CatalogGame{ID: id}
		}
		games[id] = applyOverride(base, ov)
	}

	dropped := 0
	for _, g := range games {
		if g.InstallRecipe == nil {
			dropped++
			continue
		}
		if g.Tier == "" {
			g.Tier = "compatible"
		}
		cat.Games = append(cat.Games, g)
	}
	fmt.Fprintf(os.Stderr, "dropped %d games with no install recipe\n", dropped)
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
	rel, _ := filepath.Rel(root, path)
	g := CatalogGame{
		ID:          id,
		Name:        name,
		Source:      source,
		UpstreamRef: rel,
		Summary:     truncate(egg.Description, 200),
		Protocols:   detectProtocols(egg.Startup),
		DefaultPort: detectPort(egg),
	}
	g.InstallRecipe = deriveRecipe(egg)
	if g.InstallRecipe != nil && g.InstallRecipe.Method == "steam" {
		g.SteamAppID = g.InstallRecipe.SteamAppID
	}
	return &g
}

func deriveRecipe(egg Egg) *InstallRecipe {
	script := egg.Scripts.Installation.Script
	ep := strings.ToLower(strings.TrimSpace(egg.Scripts.Installation.Entrypoint))
	if ep != "" && ep != "bash" && ep != "sh" && ep != "ash" {
		return nil
	}
	if m := steamAppRE.FindStringSubmatch(script); len(m) == 2 {
		if appid, err := strconv.Atoi(m[1]); err == nil {
			return &InstallRecipe{Method: "steam", SteamAppID: appid}
		}
	}
	for _, u := range literalURLRE.FindAllString(script, -1) {
		if archiveExtRE.MatchString(u) {
			return &InstallRecipe{Method: "downloadExtract", URL: u}
		}
	}
	return nil
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

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
