package server

import (
	"database/sql"
	"errors"
	"fmt"
)

type Server struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	GameID     string `json:"gameId"`
	Port       int    `json:"port"`
	Status     string `json:"status"`
	InstallDir string `json:"installDir"`
	StartCmd   string `json:"startCmd"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) List() ([]Server, error) {
	rows, err := s.db.Query(`SELECT id, name, game_id, port, status, install_dir, start_cmd FROM servers ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var out []Server
	for rows.Next() {
		var x Server
		if err := rows.Scan(&x.ID, &x.Name, &x.GameID, &x.Port, &x.Status, &x.InstallDir, &x.StartCmd); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	if out == nil {
		out = []Server{}
	}
	return out, rows.Err()
}

func (s *Store) Get(id int64) (*Server, error) {
	row := s.db.QueryRow(`SELECT id, name, game_id, port, status, install_dir, start_cmd FROM servers WHERE id = ?`, id)
	var x Server
	if err := row.Scan(&x.ID, &x.Name, &x.GameID, &x.Port, &x.Status, &x.InstallDir, &x.StartCmd); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &x, nil
}

func (s *Store) Create(in Server) (*Server, error) {
	res, err := s.db.Exec(
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
	return s.Get(id)
}

func (s *Store) UpdateStatus(id int64, status string) error {
	_, err := s.db.Exec(`UPDATE servers SET status = ? WHERE id = ?`, status, id)
	return err
}

func (s *Store) UpdateInstall(id int64, installDir, startCmd string) error {
	_, err := s.db.Exec(`UPDATE servers SET install_dir = ?, start_cmd = ? WHERE id = ?`, installDir, startCmd, id)
	return err
}

func (s *Store) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM servers WHERE id = ?`, id)
	return err
}
