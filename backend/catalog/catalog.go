package catalog

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed all:data
var dataFS embed.FS

type Upstream struct {
	Commit string `json:"commit,omitempty"`
	Path   string `json:"path,omitempty"`
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

var (
	byID    map[string]Game
	allList []Game
)

func Start() error {
	games, err := loadFS(dataFS, "data")
	if err != nil {
		return err
	}
	index := make(map[string]Game, len(games))
	for _, g := range games {
		index[g.ID] = g
	}
	sort.Slice(games, func(i, j int) bool {
		if games[i].Tier != games[j].Tier {
			return tierRank(games[i].Tier) < tierRank(games[j].Tier)
		}
		return games[i].Name < games[j].Name
	})
	byID = index
	allList = games
	return nil
}

func loadFS(fsys fs.FS, root string) ([]Game, error) {
	var games []Game
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".json") {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		var g Game
		if err := json.Unmarshal(data, &g); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		rel, err := fs.Sub(fsys, root)
		_ = rel
		_ = err
		// p is "<root>/<source>/<id>.json"
		parts := strings.Split(p, "/")
		if len(parts) >= 3 {
			g.Source = parts[len(parts)-2]
		}
		if g.ID == "" {
			return fmt.Errorf("%s: empty id", p)
		}
		if path.Base(p) != g.ID+".json" {
			return fmt.Errorf("%s: id %q does not match filename", p, g.ID)
		}
		games = append(games, g)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return games, nil
}

func tierRank(t string) int {
	switch t {
	case "supported":
		return 0
	case "experimental":
		return 1
	case "disabled":
		return 2
	default:
		return 3
	}
}

func All() []Game { return allList }

func Get(id string) (Game, bool) {
	g, ok := byID[id]
	return g, ok
}

func Sources() []string {
	seen := map[string]bool{}
	for _, g := range allList {
		if g.Source != "" {
			seen[g.Source] = true
		}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
