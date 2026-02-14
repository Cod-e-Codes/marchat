package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	appcfg "github.com/Cod-e-Codes/marchat/config"
)

// helper to create a temporary DB and hub for tests
func setupTestServerEnv(t *testing.T) (*sql.DB, *Hub, *appcfg.Config, func()) {
	t.Helper()
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	db.SetMaxOpenConns(1)
	CreateSchema(db)

	pluginDir := filepath.Join(tdir, "plugins")
	dataDir := filepath.Join(tdir, "data")
	_ = os.MkdirAll(pluginDir, 0o755)
	_ = os.MkdirAll(dataDir, 0o755)

	hub := NewHub(pluginDir, dataDir, "", db)
	go func() { // run hub in background
		hub.Run()
	}()

	cfg := &appcfg.Config{
		Port:              8080,
		AdminKey:          "secret-key",
		Admins:            []string{"admin"},
		DBPath:            dbPath,
		ConfigDir:         tdir,
		LogLevel:          "debug",
		PluginRegistryURL: "",
		MaxFileBytes:      1024 * 1024,
	}

	cleanup := func() {
		_ = db.Close()
	}
	return db, hub, cfg, cleanup
}

func TestAdminWeb_LoginSessionAndProtectedRoutes(t *testing.T) {
	db, hub, cfg, cleanup := setupTestServerEnv(t)
	defer cleanup()

	was := NewWebAdminServer(hub, db, cfg)
	mux := http.NewServeMux()
	was.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 1) Protected route without session should be 401
	resp, err := ts.Client().Get(ts.URL + "/admin/api/overview")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 2) Login with correct key
	loginBody, _ := json.Marshal(map[string]string{"key": cfg.AdminKey})
	resp, err = ts.Client().Post(ts.URL+"/admin/api/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on login, got %d", resp.StatusCode)
	}
	// capture session cookie
	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "admin_session" {
			sessionCookie = c
			break
		}
	}
	_ = resp.Body.Close()
	if sessionCookie == nil {
		t.Fatalf("expected admin_session cookie after login")
	}

	// 3) Session check should pass
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/admin/api/check-session", nil)
	req.AddCookie(sessionCookie)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("check-session failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on check-session, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 4) Protected overview should succeed with cookie
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/admin/api/overview", nil)
	req.AddCookie(sessionCookie)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on overview, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 5) CSRF token retrieval
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/admin/api/csrf-token", nil)
	req.AddCookie(sessionCookie)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("csrf-token failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on csrf-token, got %d", resp.StatusCode)
	}
	var csrfResp struct {
		Token string `json:"csrfToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&csrfResp); err != nil {
		_ = resp.Body.Close()
		t.Fatalf("decode csrf token: %v", err)
	}
	_ = resp.Body.Close()
	if csrfResp.Token == "" {
		t.Fatalf("empty csrf token")
	}

	// 6) System action without CSRF should be 403
	actionBody, _ := json.Marshal(map[string]string{"action": "force_gc"})
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/admin/api/action/system", bytes.NewReader(actionBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("system action (no csrf) failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 without csrf, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 7) System action with CSRF should succeed
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/admin/api/action/system", bytes.NewReader(actionBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfResp.Token)
	req.AddCookie(sessionCookie)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("system action (with csrf) failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with csrf, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// allow background goroutines to settle
	time.Sleep(50 * time.Millisecond)
}
