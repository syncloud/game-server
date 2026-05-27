package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"hooks/client"
)

type server struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	GameID     string `json:"gameId"`
	Port       int    `json:"port"`
	Status     string `json:"status"`
	InstallDir string `json:"installDir"`
	StartCmd   string `json:"startCmd"`
	LastError  string `json:"lastError,omitempty"`
}

func resolveServer(c *client.Client, ident string) (*server, error) {
	var list []server
	if err := c.Do("GET", "/api/v1/servers", nil, &list); err != nil {
		return nil, err
	}
	if id, err := strconv.ParseInt(ident, 10, 64); err == nil {
		for _, s := range list {
			if s.ID == id {
				return &s, nil
			}
		}
	}
	for _, s := range list {
		if s.Name == ident {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("no server with id/name %q", ident)
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func printServerTable(servers []server) {
	if len(servers) == 0 {
		fmt.Println("(no servers)")
		return
	}
	fmt.Printf("%-4s  %-20s  %-16s  %-6s  %s\n", "ID", "NAME", "GAME", "PORT", "STATUS")
	for _, s := range servers {
		fmt.Printf("%-4d  %-20s  %-16s  %-6d  %s\n", s.ID, s.Name, s.GameID, s.Port, s.Status)
	}
}

func serverCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{Use: "server", Short: "Manage game servers"}
	cmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "emit JSON to stdout")

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List installed game servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New()
			var list []server
			if err := c.Do("GET", "/api/v1/servers", nil, &list); err != nil {
				return err
			}
			if jsonOut {
				printJSON(list)
			} else {
				printServerTable(list)
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show <id|name>",
		Short: "Show a server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New()
			s, err := resolveServer(c, args[0])
			if err != nil {
				return err
			}
			if jsonOut {
				printJSON(s)
			} else {
				fmt.Printf("id:         %d\nname:       %s\ngame:       %s\nport:       %d\nstatus:     %s\ninstallDir: %s\nstartCmd:   %s\n",
					s.ID, s.Name, s.GameID, s.Port, s.Status, s.InstallDir, s.StartCmd)
				if s.LastError != "" {
					fmt.Printf("lastError:  %s\n", s.LastError)
				}
			}
			return nil
		},
	})

	createCmd := &cobra.Command{
		Use:   "create <name> <gameId>",
		Short: "Create a server entry (does not install)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetInt("port")
			startCmd, _ := cmd.Flags().GetString("start-cmd")
			c := client.New()
			body := map[string]any{"name": args[0], "gameId": args[1], "port": port}
			if startCmd != "" {
				body["startCmd"] = startCmd
			}
			var s server
			if err := c.Do("POST", "/api/v1/servers", body, &s); err != nil {
				return err
			}
			if jsonOut {
				printJSON(s)
			} else {
				fmt.Printf("created server[%d] %s (%s)\n", s.ID, s.Name, s.GameID)
			}
			return nil
		},
	}
	createCmd.Flags().Int("port", 0, "listen port (0 = use game default)")
	createCmd.Flags().String("start-cmd", "", "override the start command (tests / stubs)")
	cmd.AddCommand(createCmd)

	action := func(name, verb string) *cobra.Command {
		return &cobra.Command{
			Use:   name + " <id|name>",
			Short: verb + " a server",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				c := client.New()
				s, err := resolveServer(c, args[0])
				if err != nil {
					return err
				}
				var updated server
				if err := c.Do("POST", fmt.Sprintf("/api/v1/servers/%d/%s", s.ID, name), nil, &updated); err != nil {
					return err
				}
				if jsonOut {
					printJSON(updated)
				} else {
					fmt.Printf("%s server[%d] %s → %s\n", verb, updated.ID, updated.Name, updated.Status)
				}
				return nil
			},
		}
	}
	cmd.AddCommand(action("install", "installing"))
	cmd.AddCommand(action("start", "started"))
	cmd.AddCommand(action("stop", "stopped"))
	cmd.AddCommand(action("restart", "restarted"))

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <id|name>",
		Short: "Delete a server (does not remove install dir)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New()
			s, err := resolveServer(c, args[0])
			if err != nil {
				return err
			}
			return c.Do("DELETE", fmt.Sprintf("/api/v1/servers/%d", s.ID), nil, nil)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "logs <id|name>",
		Short: "Print captured stdout/stderr",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New()
			s, err := resolveServer(c, args[0])
			if err != nil {
				return err
			}
			var r struct {
				Lines []string `json:"lines"`
			}
			if err := c.Do("GET", fmt.Sprintf("/api/v1/servers/%d/logs", s.ID), nil, &r); err != nil {
				return err
			}
			if jsonOut {
				printJSON(r.Lines)
			} else {
				fmt.Print(strings.Join(r.Lines, ""))
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "query <id|name>",
		Short: "A2S_INFO query against a running Source-engine server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New()
			s, err := resolveServer(c, args[0])
			if err != nil {
				return err
			}
			var r map[string]any
			if err := c.Do("GET", fmt.Sprintf("/api/v1/servers/%d/query", s.ID), nil, &r); err != nil {
				return err
			}
			printJSON(r)
			return nil
		},
	})

	return cmd
}
