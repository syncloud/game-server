package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var cfgKVRE = regexp.MustCompile(`^(\w+)="([^"]*)"`)

var lgsmIDRemap = map[string]string{
	"cs": "hlds-cs",
	"vh": "valheim",
	"pz": "zomboid",
}

type lgsmOverride struct {
	Tier        string
	TierReason  string
	Summary     string
	Protocols   []string
	DefaultPort int
}

var lgsmOverrides = map[string]lgsmOverride{
	"hlds-cs": {
		Tier:    "verified",
		Summary: "Classic Half-Life dedicated server running CS 1.6 (~250MB). Smallest Source A2S-queryable Steam server, used as our CI Steam fixture.",
	},
	"cs2":     {Tier: "compatible", Summary: "Valve's tactical shooter dedicated server (~30GB)."},
	"tf2":     {Tier: "compatible", Summary: "Class-based team shooter (~10GB)."},
	"gmod":    {Tier: "compatible", Summary: "Sandbox modification of Source (~600MB)."},
	"valheim": {Tier: "compatible", Summary: "Viking survival co-op (~2GB)."},
	"zomboid": {Tier: "compatible", Summary: "Isometric zombie survival sandbox (~3GB).", DefaultPort: 16261},
	"ark":     {Tier: "experimental", Summary: "Dinosaur survival multiplayer (~25GB)."},
	"rust":    {Tier: "experimental", Summary: "Multiplayer survival. Requires paid Steam account.", TierReason: "Steam appid is not anonymous-loginnable"},
}

func walkLinuxGSM(root string, out map[string]CatalogGame) {
	listPath := filepath.Join(root, "lgsm", "data", "serverlist.csv")
	f, err := os.Open(listPath)
	if err != nil {
		log.Printf("linuxgsm: open %s: %v", listPath, err)
		return
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		log.Printf("linuxgsm: parse serverlist.csv: %v", err)
		return
	}

	imported, skippedNonSteam, skippedCollision := 0, 0, 0
	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue
		}
		shortname := strings.TrimSpace(row[0])
		serverName := strings.TrimSpace(row[1])
		gameName := strings.TrimSpace(row[2])
		if shortname == "" || serverName == "" {
			continue
		}

		cfg := parseLGSMCfg(filepath.Join(root, "lgsm", "config-default", "config-lgsm", serverName, "_default.cfg"))
		appid, _ := strconv.Atoi(cfg["appid"])
		if appid <= 0 {
			skippedNonSteam++
			continue
		}

		id := shortname
		if remapped, ok := lgsmIDRemap[shortname]; ok {
			id = remapped
		}
		if existing, ok := out[id]; ok {
			log.Printf("linuxgsm: skip %s (id %q already claimed by source=%s)", shortname, id, existing.Source)
			skippedCollision++
			continue
		}

		port, _ := strconv.Atoi(cfg["port"])

		g := CatalogGame{
			ID:          id,
			Name:        gameName,
			Source:      "linuxgsm",
			UpstreamRef: serverName,
			SteamAppID:  appid,
			DefaultPort: port,
			Protocols:   []string{"udp"},
			Tier:        "experimental",
			EggURL: fmt.Sprintf(
				"https://github.com/GameServerManagers/LinuxGSM/blob/%s/lgsm/config-default/config-lgsm/%s/_default.cfg",
				linuxgsmRef, serverName),
			Summary: fmt.Sprintf("Steam dedicated server for %s (appid %d). Imported from LinuxGSM.", gameName, appid),
		}
		if ov, ok := lgsmOverrides[id]; ok {
			if ov.Tier != "" {
				g.Tier = ov.Tier
			}
			if ov.TierReason != "" {
				g.TierReason = ov.TierReason
			}
			if ov.Summary != "" {
				g.Summary = ov.Summary
			}
			if len(ov.Protocols) > 0 {
				g.Protocols = ov.Protocols
			}
			if ov.DefaultPort != 0 && g.DefaultPort == 0 {
				g.DefaultPort = ov.DefaultPort
			}
		}
		out[id] = g
		imported++
	}
	fmt.Fprintf(os.Stderr, "linuxgsm: imported %d Steam games (%d non-Steam, %d ID collisions)\n",
		imported, skippedNonSteam, skippedCollision)
}

func parseLGSMCfg(path string) map[string]string {
	m := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return m
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if mm := cfgKVRE.FindStringSubmatch(line); len(mm) == 3 {
			m[mm[1]] = mm[2]
		}
	}
	return m
}
