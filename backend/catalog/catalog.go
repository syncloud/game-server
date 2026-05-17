package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed catalog.json
var raw []byte

type Game struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Source      string         `json:"source"`
	UpstreamRef string         `json:"upstreamRef,omitempty"`
	Summary     string         `json:"summary"`
	DefaultPort int            `json:"defaultPort"`
	Protocols   []string       `json:"protocols"`
	Tier        string         `json:"tier"`
	TierReason  string         `json:"tierReason,omitempty"`
	EggURL      string         `json:"eggUrl,omitempty"`
	Egg         *EggInline     `json:"egg,omitempty"`
	SteamAppID  int            `json:"steamAppId,omitempty"`
	Variables   []EnvVariable  `json:"variables,omitempty"`
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

type bundle struct {
	Sources map[string]string `json:"sources"`
	Games   []Game            `json:"games"`
}

var (
	loaded  bundle
	byID    map[string]Game
	allList []Game
)

func Start() error {
	var b bundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return fmt.Errorf("parse embedded catalog.json: %w", err)
	}
	b.Games = append(b.Games, steamEntries()...)
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
	loaded = b
	byID = index
	allList = list
	return nil
}

func tierRank(t string) int {
	switch t {
	case "verified":
		return 0
	case "compatible":
		return 1
	case "experimental":
		return 2
	default:
		return 3
	}
}

func All() []Game        { return allList }
func Sources() map[string]string { return loaded.Sources }
func Get(id string) (Game, bool) {
	g, ok := byID[id]
	return g, ok
}
