package db

import (
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/syncloud/games/backend/server"
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
	path string
	conn *sql.DB
}

func New(path string) *DB {
	return &DB{path: path}
}

func (d *DB) Start() error {
	conn, err := sql.Open("sqlite", d.path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return fmt.Errorf("open %s: %w", d.path, err)
	}
	if _, err := conn.Exec(createSchema); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	_, _ = conn.Exec(`ALTER TABLE servers ADD COLUMN last_error TEXT`)
	d.conn = conn
	return nil
}

func (d *DB) Close() error {
	if d.conn == nil {
		return nil
	}
	return d.conn.Close()
}

func (d *DB) ListServers() ([]server.Server, error) {
	rows, err := d.conn.Query(`SELECT id, name, game_id, port, status, install_dir, start_cmd, COALESCE(last_error, '') FROM servers ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var out []server.Server
	for rows.Next() {
		var x server.Server
		if err := rows.Scan(&x.ID, &x.Name, &x.GameID, &x.Port, &x.Status, &x.InstallDir, &x.StartCmd, &x.LastError); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	if out == nil {
		out = []server.Server{}
	}
	return out, rows.Err()
}

func (d *DB) GetServer(id int64) (*server.Server, error) {
	row := d.conn.QueryRow(`SELECT id, name, game_id, port, status, install_dir, start_cmd, COALESCE(last_error, '') FROM servers WHERE id = ?`, id)
	var x server.Server
	if err := row.Scan(&x.ID, &x.Name, &x.GameID, &x.Port, &x.Status, &x.InstallDir, &x.StartCmd, &x.LastError); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &x, nil
}

func (d *DB) CreateServer(in server.Server) (*server.Server, error) {
	res, err := d.conn.Exec(
		`INSERT INTO servers (name, game_id, port, status, install_dir, start_cmd) VALUES (?, ?, ?, ?, ?, ?)`,
		in.Name, in.GameID, in.Port, "stopped", in.InstallDir, in.StartCmd,
	)
	if err != nil {
		return nil, fmt.Errorf("insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return d.GetServer(id)
}

func (d *DB) UpdateServerStatus(id int64, status string) error {
	_, err := d.conn.Exec(`UPDATE servers SET status = ? WHERE id = ?`, status, id)
	return err
}

func (d *DB) UpdateServerInstall(id int64, installDir, startCmd string) error {
	_, err := d.conn.Exec(`UPDATE servers SET install_dir = ?, start_cmd = ?, last_error = NULL WHERE id = ?`, installDir, startCmd, id)
	return err
}

func (d *DB) UpdateServerLastError(id int64, msg string) error {
	_, err := d.conn.Exec(`UPDATE servers SET last_error = ? WHERE id = ?`, msg, id)
	return err
}

func (d *DB) DeleteServer(id int64) error {
	_, err := d.conn.Exec(`DELETE FROM servers WHERE id = ?`, id)
	return err
}
