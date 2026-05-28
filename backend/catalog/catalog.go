package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed catalog.json
var raw []byte

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

type bundle struct {
	Games []Game `json:"games"`
}

var (
	byID    map[string]Game
	allList []Game
)

func Start() error {
	var b bundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return fmt.Errorf("parse embedded catalog.json: %w", err)
	}
	index := make(map[string]Game, len(b.Games))
	for _, g := range b.Games {
		index[g.ID] = g
	}
	list := append([]Game(nil), b.Games...)
	sort.Slice(list, func(i, j int) bool {
		if list[i].Tier != list[j].Tier {
			return tierRank(list[i].Tier) < tierRank(list[j].Tier)
		}
		return list[i].Name < list[j].Name
	})
	byID = index
	allList = list
	return nil
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
