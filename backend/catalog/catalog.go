// catalog: the game registry. Generated at build time from
// parkervcp/eggs and pelican-eggs/games (see catalog/build.sh), plus
// hand-curated Steam entries (steam.go).
package catalog

import (
	_ "embed"
	"encoding/json"
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

func init() {
	if err := json.Unmarshal(raw, &loaded); err != nil {
		panic("catalog: parse embedded catalog.json: " + err.Error())
	}
	// merge curated Steam entries
	loaded.Games = append(loaded.Games, steamEntries()...)
	byID = map[string]Game{}
	for _, g := range loaded.Games {
		byID[g.ID] = g
	}
	allList = append([]Game(nil), loaded.Games...)
	sort.Slice(allList, func(i, j int) bool {
		if allList[i].Tier != allList[j].Tier {
			return tierRank(allList[i].Tier) < tierRank(allList[j].Tier)
		}
		return allList[i].Name < allList[j].Name
	})
}

func tierRank(t string) int {
	switch t {
	case "supported":
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
