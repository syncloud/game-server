package server

type Server struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	GameID     string `json:"gameId"`
	GameName   string `json:"gameName"`
	Port       int    `json:"port"`
	Status     string `json:"status"`
	InstallDir string `json:"installDir"`
	StartCmd   string `json:"startCmd"`
	LocalIp    string `json:"localIp"`
	LastError  string `json:"lastError,omitempty"`
}
