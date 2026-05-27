package server

type Server struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	GameID     string `json:"gameId"`
	Port       int    `json:"port"`
	Status     string `json:"status"`
	InstallDir string `json:"installDir"`
	StartCmd   string `json:"startCmd"`
	LastError  string `json:"lastError,omitempty"`
}
