package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/syncloud/games/backend/api"
	"github.com/syncloud/games/backend/auth"
	"github.com/syncloud/games/backend/catalog"
	"github.com/syncloud/games/backend/db"
	"github.com/syncloud/games/backend/installer"
	"github.com/syncloud/games/backend/runner"
)

const (
	oidcConfigPath = "/var/snap/games/current/oidc.json"
	dbPath         = "/var/snap/games/current/database.db"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	root := &cobra.Command{
		Use:           "backend",
		Short:         "Games backend HTTP service",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(logger)
		},
	}
	if err := root.Execute(); err != nil {
		logger.Fatal("backend", zap.Error(err))
	}
}

func run(logger *zap.Logger) error {
	if err := catalog.Start(); err != nil {
		return fmt.Errorf("catalog: %w", err)
	}
	store, err := openDB(logger)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	defer store.Close()

	stdLogger := zap.NewStdLog(logger)
	proc := runner.New(stdLogger)
	inst := installer.New(
		installer.ServersBaseDir,
		installer.NewRecipeInstaller(installer.SteamCMDPath, installer.SteamLib32, installer.SteamLib64),
	)
	authService := loadAuth(logger)

	return api.New(logger, store, proc, inst, authService).Start()
}

type oidcFileConfig struct {
	AuthUrl      string `json:"authUrl"`
	AuthSocket   string `json:"authSocket"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	RedirectUrl  string `json:"redirectUrl"`
}

func loadAuth(logger *zap.Logger) *auth.Service {
	data, err := os.ReadFile(oidcConfigPath)
	if err != nil {
		logger.Info("auth: oidc.json missing; /api/ will be unprotected", zap.Error(err))
		return nil
	}
	var c oidcFileConfig
	if err := json.Unmarshal(data, &c); err != nil {
		logger.Warn("auth: oidc.json parse", zap.Error(err))
		return nil
	}
	if c.AuthSocket == "" {
		logger.Warn("auth: oidc.json missing authSocket; /api/ will be unprotected")
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	svc, err := auth.NewService(ctx, zap.NewStdLog(logger), c.AuthUrl, c.AuthSocket, c.ClientID, c.ClientSecret, c.ClientSecret, c.RedirectUrl)
	if err != nil {
		logger.Warn("auth: init", zap.Error(err))
		return nil
	}
	logger.Info("auth: OIDC ready",
		zap.String("provider", c.AuthUrl), zap.String("socket", c.AuthSocket),
		zap.String("client", c.ClientID), zap.String("redirect", c.RedirectUrl))
	return svc
}

func openDB(logger *zap.Logger) (*db.DB, error) {
	d := db.New(dbPath)
	if err := d.Start(); err != nil {
		if _, statErr := os.Stat("/var/snap/games/current"); statErr != nil {
			logger.Warn("data dir missing, falling back to in-memory db", zap.Error(statErr))
			d = db.New(":memory:")
			if err := d.Start(); err != nil {
				return nil, err
			}
			return d, nil
		}
		return nil, err
	}
	return d, nil
}
