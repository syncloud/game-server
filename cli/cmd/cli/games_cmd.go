package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"hooks/client"
)

type catalogEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Source      string   `json:"source"`
	UpstreamRef string   `json:"upstreamRef"`
	Tier        string   `json:"tier"`
	DefaultPort int      `json:"defaultPort"`
	Protocols   []string `json:"protocols"`
}

func gamesCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{Use: "games", Short: "Browse the game catalog"}
	cmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "emit JSON to stdout")

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all games in the catalog",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New()
			var list []catalogEntry
			if err := c.Do("GET", "/api/v1/games", nil, &list); err != nil {
				return err
			}
			if jsonOut {
				printJSON(list)
			} else {
				fmt.Printf("%-26s  %-12s  %-13s  %-6s  %s\n", "ID", "SOURCE", "TIER", "PORT", "NAME")
				for _, g := range list {
					fmt.Printf("%-26s  %-12s  %-13s  %-6d  %s\n", g.ID, g.Source, g.Tier, g.DefaultPort, g.Name)
				}
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "sources",
		Short: "Pinned upstream egg-catalog SHAs",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New()
			var r map[string]string
			if err := c.Do("GET", "/api/v1/catalog/sources", nil, &r); err != nil {
				return err
			}
			if jsonOut {
				printJSON(r)
			} else {
				for k, v := range r {
					fmt.Printf("%-24s  %s\n", k, v)
				}
			}
			return nil
		},
	})

	return cmd
}

func healthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check backend health",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New()
			var r map[string]string
			if err := c.Do("GET", "/api/v1/health", nil, &r); err != nil {
				return err
			}
			fmt.Println(r["status"])
			return nil
		},
	}
}
