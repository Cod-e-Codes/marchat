package server

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/gorilla/websocket"
)

func TestPermanentBanPersistsAcrossHubRestart(t *testing.T) {
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "moderation.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	CreateSchema(db)

	hub1 := mustNewHub(t, tdir, tdir, "", db)
	if err := hub1.BanUser("alice", "admin"); err != nil {
		t.Fatalf("BanUser: %v", err)
	}
	if !hub1.IsUserBanned("alice") {
		t.Fatal("alice should be banned before restart")
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB reopen: %v", err)
	}
	defer db2.Close()
	CreateSchema(db2)

	hub2 := mustNewHub(t, tdir, tdir, "", db2)
	if !hub2.IsUserBanned("alice") {
		t.Fatal("alice should still be banned after hub restart")
	}
	if !hub2.IsUserBanned("ALICE") {
		t.Fatal("persisted ban should remain case-insensitive")
	}

	assertHandshakeRejectedAsBanned(t, hub2, db2, tdir, "alice")
}

func TestTempKickPersistsAcrossHubRestart(t *testing.T) {
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "moderation.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	CreateSchema(db)

	hub1 := mustNewHub(t, tdir, tdir, "", db)
	registerTestClient(hub1, "bob")
	if err := hub1.KickUser("bob", "admin"); err != nil {
		t.Fatalf("KickUser: %v", err)
	}
	if !hub1.IsUserBanned("bob") {
		t.Fatal("bob should be kicked before restart")
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB reopen: %v", err)
	}
	defer db2.Close()
	CreateSchema(db2)

	hub2 := mustNewHub(t, tdir, tdir, "", db2)
	if !hub2.IsUserBanned("bob") {
		t.Fatal("bob should still be kicked after hub restart")
	}

	hub2.banMutex.RLock()
	_, permanent := hub2.bans["bob"]
	kickExpiry, kicked := hub2.tempKicks["bob"]
	hub2.banMutex.RUnlock()
	if permanent {
		t.Fatal("temp kick must not load as permanent ban")
	}
	if !kicked || !time.Now().Before(kickExpiry) {
		t.Fatalf("expected unexpired tempKick, got kicked=%v expiry=%v", kicked, kickExpiry)
	}

	assertHandshakeRejectedAsBanned(t, hub2, db2, tdir, "bob")
}

func TestExpiredTempKickNotLoadedOnRestart(t *testing.T) {
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "moderation.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	CreateSchema(db)

	past := time.Now().Add(-time.Hour)
	if err := recordBanEvent(db, "carol", "admin", &past); err != nil {
		t.Fatalf("recordBanEvent: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB reopen: %v", err)
	}
	defer db2.Close()
	CreateSchema(db2)

	hub := mustNewHub(t, tdir, tdir, "", db2)
	if hub.IsUserBanned("carol") {
		t.Fatal("expired temp kick must not reject after restart")
	}
}

func assertHandshakeRejectedAsBanned(t *testing.T, hub *Hub, db interface {
	Close() error
}, tdir, username string) {
	t.Helper()
	_ = db
	sqlDB := hub.getDB()
	go hub.Run()

	handler := ServeWs(hub, sqlDB, nil, "admin-key", false, 10<<20, filepath.Join(tdir, "ws.db"))
	srv := httptest.NewServer(handler)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(shared.Handshake{Username: username}); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatal("expected close after ban reject, got a message")
	}
	closeErr, ok := err.(*websocket.CloseError)
	if !ok {
		if !strings.Contains(strings.ToLower(err.Error()), "banned") {
			t.Fatalf("expected ban close, got %T: %v", err, err)
		}
		return
	}
	if closeErr.Code != websocket.ClosePolicyViolation {
		t.Fatalf("expected ClosePolicyViolation, got %d (%s)", closeErr.Code, closeErr.Text)
	}
	if !strings.Contains(strings.ToLower(closeErr.Text), "banned") {
		t.Fatalf("expected ban close text, got %q", closeErr.Text)
	}
}
