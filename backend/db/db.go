package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const createSchema = `
CREATE TABLE IF NOT EXISTS servers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  game_id TEXT NOT NULL,
  port INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'stopped',
  install_dir TEXT NOT NULL DEFAULT '',
  start_cmd TEXT NOT NULL DEFAULT '',
  steam_user TEXT NOT NULL DEFAULT '',
  steam_pass TEXT NOT NULL DEFAULT '',
  last_error TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	if _, err := conn.Exec(createSchema); err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}
	// idempotent migration for snaps upgraded from a schema without last_error
	_, _ = conn.Exec(`ALTER TABLE servers ADD COLUMN last_error TEXT`)
	return &DB{conn}, nil
}
