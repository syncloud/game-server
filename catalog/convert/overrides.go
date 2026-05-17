package main

import (
	"encoding/json"
	"log"
	"os"
)

type Override struct {
	Source        *string        `json:"source,omitempty"`
	Name          *string        `json:"name,omitempty"`
	Summary       *string        `json:"summary,omitempty"`
	Tier          *string        `json:"tier,omitempty"`
	TierReason    *string        `json:"tierReason,omitempty"`
	DefaultPort   *int           `json:"defaultPort,omitempty"`
	Protocols     []string       `json:"protocols,omitempty"`
	SteamAppID    *int           `json:"steamAppId,omitempty"`
	InstallRecipe *InstallRecipe `json:"installRecipe,omitempty"`
	Start         *StartRecipe   `json:"start,omitempty"`
}

func loadOverrides(path string) map[string]Override {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("overrides: open %s: %v", path, err)
	}
	var m map[string]Override
	if err := json.Unmarshal(data, &m); err != nil {
		log.Fatalf("overrides: parse %s: %v", path, err)
	}
	return m
}

func applyOverride(g CatalogGame, o Override) CatalogGame {
	if o.Source != nil {
		g.Source = *o.Source
	}
	if o.Name != nil {
		g.Name = *o.Name
	}
	if o.Summary != nil {
		g.Summary = *o.Summary
	}
	if o.Tier != nil {
		g.Tier = *o.Tier
	}
	if o.TierReason != nil {
		g.TierReason = *o.TierReason
	}
	if o.DefaultPort != nil {
		g.DefaultPort = *o.DefaultPort
	}
	if len(o.Protocols) > 0 {
		g.Protocols = o.Protocols
	}
	if o.SteamAppID != nil {
		g.SteamAppID = *o.SteamAppID
	}
	if o.InstallRecipe != nil {
		g.InstallRecipe = o.InstallRecipe
	}
	if o.Start != nil {
		g.Start = o.Start
	}
	return g
}
