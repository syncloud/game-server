package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Upstream struct {
	Commit string `json:"commit,omitempty"`
	Path   string `json:"path,omitempty"`
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

type Game struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Source         string         `json:"source"`
	Summary        string         `json:"summary"`
	Tier           string         `json:"tier"`
	DisabledReason string         `json:"disabledReason,omitempty"`
	DefaultPort    int            `json:"defaultPort"`
	Protocols      []string       `json:"protocols"`
	Upstream       *Upstream      `json:"upstream,omitempty"`
	InstallRecipe  *InstallRecipe `json:"installRecipe,omitempty"`
	Start          *StartRecipe   `json:"start,omitempty"`
}

type Bundle struct {
	Games []Game `json:"games"`
}

var validTiers = map[string]bool{
	"supported":    true,
	"experimental": true,
	"disabled":     true,
}

func main() {
	root := flag.String("root", "catalog", "directory containing source-named subdirs of *.json")
	out := flag.String("out", "", "output catalog.json path")
	flag.Parse()
	if *out == "" {
		log.Fatal("--out required")
	}

	games := []Game{}
	seen := map[string]string{}

	err := filepath.WalkDir(*root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		rel, _ := filepath.Rel(*root, path)
		parts := strings.SplitN(rel, string(filepath.Separator), 2)
		if len(parts) != 2 {
			return nil
		}
		source := parts[0]
		fileID := strings.TrimSuffix(filepath.Base(path), ".json")

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		var g Game
		dec := json.NewDecoder(strings.NewReader(string(data)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&g); err != nil {
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		if g.ID != fileID {
			return fmt.Errorf("%s: id %q does not match filename %q", rel, g.ID, fileID)
		}
		if !validTiers[g.Tier] {
			return fmt.Errorf("%s: tier %q must be supported|experimental|disabled", rel, g.Tier)
		}
		if g.Tier == "disabled" {
			if g.DisabledReason == "" {
				return fmt.Errorf("%s: tier=disabled requires disabledReason", rel)
			}
		} else {
			if g.InstallRecipe == nil {
				return fmt.Errorf("%s: tier=%s requires installRecipe", rel, g.Tier)
			}
			if g.InstallRecipe.Method == "" {
				return fmt.Errorf("%s: installRecipe.method required", rel)
			}
		}
		if other, dup := seen[g.ID]; dup {
			return fmt.Errorf("duplicate id %q in %s and %s", g.ID, other, rel)
		}
		seen[g.ID] = rel
		g.Source = source
		games = append(games, g)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	sort.Slice(games, func(i, j int) bool { return games[i].ID < games[j].ID })

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(Bundle{Games: games}); err != nil {
		log.Fatal(err)
	}
	f.Close()

	tiers := map[string]int{}
	for _, g := range games {
		tiers[g.Tier]++
	}
	fmt.Fprintf(os.Stderr, "wrote %d games to %s — by tier: %v\n", len(games), *out, tiers)
}
