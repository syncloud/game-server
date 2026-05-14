package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const SyncloudCAPath = "/var/snap/platform/current/syncloud.ca.crt"

const (
	SessionCookie = "games_session"
	StateCookie   = "games_oauth_state"
	VerifierCookie = "games_oauth_verifier"
	SessionTTL    = 24 * time.Hour
)

type User struct {
	Sub   string `json:"sub"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type Service struct {
	mu           sync.Mutex
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	cfg          oauth2.Config
	signKey      []byte
	authzAPI     string // for HTTP-Basic delegated check, optional
	tlsClient    *http.Client
	authUrl      string
	logger       *log.Logger
}

type ctxKey struct{}

func ContextUser(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(ctxKey{}).(*User)
	return u, ok
}

func NewService(ctx context.Context, logger *log.Logger, authUrl, clientID, clientSecret, signSecret, redirectURL string) (*Service, error) {
	// Authelia ships with a self-signed Syncloud CA. The configure hook
	// drops it into /usr/local/share/ca-certificates/ + update-ca-certificates
	// so the OS trust store covers us; this is the in-process safety net
	// for first-boot races and for when the hook copy gets skipped.
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if pem, err := os.ReadFile(SyncloudCAPath); err == nil {
		pool.AppendCertsFromPEM(pem)
	} else {
		logger.Printf("auth: syncloud CA not readable at %s: %v", SyncloudCAPath, err)
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}
	hc := &http.Client{Transport: tr, Timeout: 15 * time.Second}
	ctx = oidc.ClientContext(ctx, hc)

	provider, err := oidc.NewProvider(ctx, authUrl)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery against %s: %w", authUrl, err)
	}
	cfg := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	key := sha256.Sum256([]byte(signSecret + "|games-session-v1"))
	return &Service{
		provider:  provider,
		verifier:  verifier,
		cfg:       cfg,
		signKey:   key[:],
		tlsClient: hc,
		authUrl:   authUrl,
		logger:    logger,
	}, nil
}

func (s *Service) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state := randToken(24)
	verifier := randToken(32)
	challenge := pkceChallenge(verifier)
	http.SetCookie(w, &http.Cookie{Name: StateCookie, Value: state, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 600})
	http.SetCookie(w, &http.Cookie{Name: VerifierCookie, Value: verifier, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 600})

	url := s.cfg.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Service) HandleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(StateCookie)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		s.logger.Printf("auth callback: state mismatch")
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	verifierCookie, err := r.Cookie(VerifierCookie)
	if err != nil {
		http.Error(w, "missing verifier", http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	ctx := oidc.ClientContext(r.Context(), s.tlsClient)
	token, err := s.cfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifierCookie.Value))
	if err != nil {
		s.logger.Printf("auth callback: token exchange: %v", err)
		http.Error(w, "token exchange failed", http.StatusUnauthorized)
		return
	}
	rawID, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token in response", http.StatusUnauthorized)
		return
	}
	idTok, err := s.verifier.Verify(ctx, rawID)
	if err != nil {
		s.logger.Printf("auth callback: id_token verify: %v", err)
		http.Error(w, "id_token verify failed", http.StatusUnauthorized)
		return
	}
	var claims struct {
		Sub   string `json:"sub"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := idTok.Claims(&claims); err != nil {
		http.Error(w, "claims decode failed", http.StatusInternalServerError)
		return
	}

	cookie, err := s.mintSession(User{Sub: claims.Sub, Name: claims.Name, Email: claims.Email})
	if err != nil {
		http.Error(w, "session mint failed", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, cookie)
	http.SetCookie(w, &http.Cookie{Name: StateCookie, Value: "", Path: "/", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: VerifierCookie, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Service) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Service) HandleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := ContextUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(u)
}

// Middleware enforces auth on the wrapped handler. Accepts:
// 1. games_session cookie (set after OIDC login)
// 2. HTTP Basic Auth — delegated to Authelia's /api/authz/auth-request
//    endpoint for validation. Lets integration tests + curl users in.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u := s.userFromCookie(r); u != nil {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
			return
		}
		if u := s.userFromBasic(r); u != nil {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
			return
		}
		// For SPA users: redirect to login. For API clients: 401.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("WWW-Authenticate", `Basic realm="games"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.Redirect(w, r, "/auth/login", http.StatusFound)
	})
}

func (s *Service) userFromCookie(r *http.Request) *User {
	c, err := r.Cookie(SessionCookie)
	if err != nil {
		return nil
	}
	u, err := s.verifySession(c.Value)
	if err != nil {
		return nil
	}
	return u
}

// userFromBasic delegates to Authelia's auth-request endpoint to validate
// the Basic credentials. Authelia checks against the platform LDAP and
// returns 200 with Remote-User on success.
func (s *Service) userFromBasic(r *http.Request) *User {
	user, pass, ok := r.BasicAuth()
	if !ok || user == "" {
		return nil
	}
	// Authelia's auth-request endpoint is /api/authz/auth-request — the
	// `/basic` we used before came from the nginx location *name* in the
	// old forward-auth conf, not the upstream path. With Basic Auth set
	// on the request, Authelia validates against LDAP and returns 200 +
	// Remote-User / Remote-Email / Remote-Name response headers.
	req, _ := http.NewRequest("GET", s.authUrl+"/api/authz/auth-request", nil)
	req.SetBasicAuth(user, pass)
	req.Header.Set("X-Original-Method", r.Method)
	req.Header.Set("X-Original-URL", "https://"+r.Host+r.URL.Path)
	req.Header.Set("X-Forwarded-Method", r.Method)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", r.Host)
	req.Header.Set("X-Forwarded-Uri", r.URL.Path)
	resp, err := s.tlsClient.Do(req)
	if err != nil {
		s.logger.Printf("auth basic delegate to authelia: %v", err)
		return nil
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 200 {
		return nil
	}
	return &User{
		Sub:   resp.Header.Get("Remote-User"),
		Name:  resp.Header.Get("Remote-Name"),
		Email: resp.Header.Get("Remote-Email"),
	}
}

type sessionPayload struct {
	User
	Exp int64 `json:"exp"`
}

func (s *Service) mintSession(u User) (*http.Cookie, error) {
	p := sessionPayload{User: u, Exp: time.Now().Add(SessionTTL).Unix()}
	body, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	b64 := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, s.signKey)
	mac.Write([]byte(b64))
	sig := hex.EncodeToString(mac.Sum(nil))
	value := b64 + "." + sig
	return &http.Cookie{
		Name:     SessionCookie,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(SessionTTL.Seconds()),
	}, nil
}

func (s *Service) verifySession(value string) (*User, error) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("bad session shape")
	}
	mac := hmac.New(sha256.New, s.signKey)
	mac.Write([]byte(parts[0]))
	expectSig := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectSig), []byte(parts[1])) {
		return nil, errors.New("session signature mismatch")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var p sessionPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, err
	}
	if time.Now().Unix() > p.Exp {
		return nil, errors.New("session expired")
	}
	return &p.User, nil
}

func randToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func pkceChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// LoadClientSecret reads the OIDC client_secret persisted by the cli's
// configure hook at /var/snap/games/current/oidc.secret.
func LoadClientSecret(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
