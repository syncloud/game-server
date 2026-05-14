package catalog

func steamEntries() []Game {
	return []Game{
		{
			ID: "hlds-cs", Name: "Counter-Strike 1.6 (HLDS)",
			Source: "steam", SteamAppID: 90,
			Summary:     "Classic Half-Life dedicated server running CS 1.6 (~250MB). Smallest Source A2S-queryable Steam server, used as our CI Steam fixture.",
			DefaultPort: 27015, Protocols: []string{"udp"},
			Tier: "verified",
		},
		{
			ID: "cs2", Name: "Counter-Strike 2",
			Source: "steam", SteamAppID: 730,
			Summary:     "Valve's tactical shooter dedicated server (~30GB).",
			DefaultPort: 27015, Protocols: []string{"udp"},
			Tier: "compatible",
		},
		{
			ID: "tf2", Name: "Team Fortress 2",
			Source: "steam", SteamAppID: 232250,
			Summary:     "Class-based team shooter (~10GB).",
			DefaultPort: 27015, Protocols: []string{"udp"},
			Tier: "compatible",
		},
		{
			ID: "gmod", Name: "Garry's Mod",
			Source: "steam", SteamAppID: 4020,
			Summary:     "Sandbox modification of Source (~600MB).",
			DefaultPort: 27015, Protocols: []string{"udp"},
			Tier: "compatible",
		},
		{
			ID: "valheim", Name: "Valheim",
			Source: "steam", SteamAppID: 896660,
			Summary:     "Viking survival co-op (~2GB).",
			DefaultPort: 2456, Protocols: []string{"udp"},
			Tier: "compatible",
		},
		{
			ID: "zomboid", Name: "Project Zomboid",
			Source: "steam", SteamAppID: 380870,
			Summary:     "Isometric zombie survival sandbox (~3GB).",
			DefaultPort: 16261, Protocols: []string{"udp"},
			Tier: "compatible",
		},
		{
			ID: "ark", Name: "ARK: Survival Evolved",
			Source: "steam", SteamAppID: 376030,
			Summary:     "Dinosaur survival multiplayer (~25GB).",
			DefaultPort: 7777, Protocols: []string{"udp"},
			Tier: "experimental",
		},
		{
			ID: "rust", Name: "Rust",
			Source: "steam", SteamAppID: 258550,
			Summary:     "Multiplayer survival. Requires paid Steam account.",
			DefaultPort: 28015, Protocols: []string{"udp"},
			Tier: "experimental",
		},
	}
}
