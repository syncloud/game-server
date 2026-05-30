package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/syncloud/games/backend/catalog"
	"github.com/syncloud/games/backend/db"
	"github.com/syncloud/games/backend/server"
)

func testApi(t *testing.T) *Api {
	t.Helper()
	if err := catalog.Start(); err != nil {
		t.Fatalf("catalog: %v", err)
	}
	d := db.New(":memory:")
	if err := d.Start(); err != nil {
		t.Fatalf("db start: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return New(zap.NewNop(), d, nil, nil, nil)
}

func postServer(t *testing.T, a *Api, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers", bytes.NewReader(raw))
	a.handleServers(rec, req)
	return rec
}

func TestCreateDerivesNameAndEnrichesGameName(t *testing.T) {
	a := testApi(t)

	rec := postServer(t, a, map[string]any{"gameId": "teeworlds", "port": 8303})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: want 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var s server.Server
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.Name != "teeworlds" {
		t.Errorf("name should default to gameId, got %q", s.Name)
	}
	if s.GameName == "" {
		t.Errorf("gameName should be enriched from catalog, got empty")
	}
}

func TestCreateKeepsExplicitName(t *testing.T) {
	a := testApi(t)

	rec := postServer(t, a, map[string]any{"name": "my-tw", "gameId": "teeworlds"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var s server.Server
	_ = json.Unmarshal(rec.Body.Bytes(), &s)
	if s.Name != "my-tw" {
		t.Errorf("explicit name should be kept, got %q", s.Name)
	}
}

func TestCreateOnePerGameRejectsDuplicate(t *testing.T) {
	a := testApi(t)

	if rec := postServer(t, a, map[string]any{"gameId": "teeworlds"}); rec.Code != http.StatusCreated {
		t.Fatalf("first create: want 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	rec := postServer(t, a, map[string]any{"gameId": "teeworlds"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate game: want 409, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "already installed") {
		t.Errorf("409 body should explain duplicate: %s", rec.Body.String())
	}
}

func TestCreateUnknownGameRejected(t *testing.T) {
	a := testApi(t)

	rec := postServer(t, a, map[string]any{"gameId": "not-a-game"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "unknown gameId") {
		t.Errorf("body should say unknown gameId: %s", rec.Body.String())
	}
}

func TestListEnrichesGameName(t *testing.T) {
	a := testApi(t)
	if rec := postServer(t, a, map[string]any{"gameId": "teeworlds"}); rec.Code != http.StatusCreated {
		t.Fatalf("create: %d (%s)", rec.Code, rec.Body.String())
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	a.handleServers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: want 200, got %d", rec.Code)
	}
	var list []server.Server
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 server, got %d", len(list))
	}
	if list[0].GameName == "" {
		t.Errorf("list should enrich gameName")
	}
}
