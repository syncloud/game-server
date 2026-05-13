// steam handles paid Steam account login via the bundled steamcmd. The
// password is never persisted — after a successful login steamcmd writes
// a sentry file to $HOME/Steam/ which acts as a long-lived session token;
// future installs use \`+login <username>\` and pick up the cached sentry.
package steam

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	SteamCMDPath = "/snap/games/current/bin/steamcmd.sh"
	UsernameFile = "/var/snap/games/current/steam-username"
)

var (
	ErrInvalidCredentials = errors.New("invalid Steam credentials")
	ErrNeedsGuardCode     = errors.New("Steam Guard code required")
)

type LoginResult struct {
	OK       bool
	Username string
	Needs2FA bool
	Prompt   string
	Raw      string
}

func Login(ctx context.Context, username, password, guardCode string) (*LoginResult, error) {
	if strings.TrimSpace(username) == "" {
		return nil, errors.New("username required")
	}

	args := []string{"+login", username, password}
	if strings.TrimSpace(guardCode) != "" {
		args = append(args, guardCode)
	}
	args = append(args, "+quit")

	ctx2, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx2, SteamCMDPath, args...)
	out, _ := cmd.CombinedOutput()
	raw := string(out)
	low := strings.ToLower(raw)

	res := &LoginResult{Username: username, Raw: raw}

	if strings.Contains(low, "logon success") ||
		strings.Contains(low, "logged in ok") ||
		(strings.Contains(low, "loading steam api...ok") && strings.Contains(low, "connecting anonymously") == false && cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 0) {
		res.OK = true
		if err := persistUsername(username); err != nil {
			return res, err
		}
		return res, nil
	}

	if strings.Contains(low, "steam guard code") ||
		strings.Contains(low, "two-factor code") ||
		strings.Contains(low, "auth code") ||
		strings.Contains(low, "rate limit exceeded") == false && strings.Contains(low, "guard") {
		res.Needs2FA = true
		res.Prompt = "Steam Guard code required — check your email or Steam mobile app"
		return res, nil
	}

	if strings.Contains(low, "invalid password") ||
		strings.Contains(low, "login failure") ||
		strings.Contains(low, "incorrect password") {
		return res, ErrInvalidCredentials
	}

	// Treat unknown failure as invalid creds rather than success — safer.
	return res, errors.New("steamcmd login failed (unknown response)")
}

func StoredUsername() string {
	b, err := os.ReadFile(UsernameFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func persistUsername(u string) error {
	if err := os.MkdirAll(filepath.Dir(UsernameFile), 0755); err != nil {
		return err
	}
	return os.WriteFile(UsernameFile, []byte(strings.TrimSpace(u)+"\n"), 0640)
}
