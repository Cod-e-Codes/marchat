package server

import (
	"database/sql"
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
	if err := CloseDB(db); err != nil {
		t.Fatalf("close db: %v", err)
	}

	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB reopen: %v", err)
	}
	defer CloseDB(db2)
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
	if err := CloseDB(db); err != nil {
		t.Fatalf("close db: %v", err)
	}

	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB reopen: %v", err)
	}
	defer CloseDB(db2)
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
	if err := CloseDB(db); err != nil {
		t.Fatalf("close db: %v", err)
	}

	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB reopen: %v", err)
	}
	defer CloseDB(db2)
	CreateSchema(db2)

	hub := mustNewHub(t, tdir, tdir, "", db2)
	if hub.IsUserBanned("carol") {
		t.Fatal("expired temp kick must not reject after restart")
	}
}

func countOpenBanRows(t *testing.T, db *sql.DB, username string) int {
	t.Helper()
	var n int
	if err := dbQueryRow(db, `SELECT COUNT(*) FROM ban_history WHERE username = ? AND unbanned_at IS NULL`, strings.ToLower(username)).Scan(&n); err != nil {
		t.Fatalf("count open ban rows: %v", err)
	}
	return n
}

func TestKickThenBanLeavesSingleOpenRow(t *testing.T) {
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "moderation.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	CreateSchema(db)

	hub := mustNewHub(t, tdir, tdir, "", db)
	registerTestClient(hub, "bob")
	if err := hub.KickUser("bob", "admin"); err != nil {
		t.Fatalf("KickUser: %v", err)
	}
	if n := countOpenBanRows(t, db, "bob"); n != 1 {
		t.Fatalf("after kick open rows = %d, want 1", n)
	}

	if err := hub.BanUser("bob", "admin"); err != nil {
		t.Fatalf("BanUser: %v", err)
	}
	if n := countOpenBanRows(t, db, "bob"); n != 1 {
		t.Fatalf("after kick+ban open rows = %d, want 1", n)
	}

	var expires sql.NullTime
	if err := dbQueryRow(db, `SELECT expires_at FROM ban_history WHERE username = ? AND unbanned_at IS NULL`, "bob").Scan(&expires); err != nil {
		t.Fatalf("scan open row: %v", err)
	}
	if expires.Valid {
		t.Fatal("open row after permanent ban should have NULL expires_at")
	}

	if err := CloseDB(db); err != nil {
		t.Fatalf("close: %v", err)
	}
	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer CloseDB(db2)
	CreateSchema(db2)
	hub2 := mustNewHub(t, tdir, tdir, "", db2)

	if !hub2.IsUserBanned("bob") {
		t.Fatal("bob should be permanently banned after restart")
	}
	hub2.banMutex.RLock()
	_, permanent := hub2.bans["bob"]
	_, kicked := hub2.tempKicks["bob"]
	hub2.banMutex.RUnlock()
	if !permanent {
		t.Fatal("expected bans map entry after kick-then-ban restart")
	}
	if kicked {
		t.Fatal("tempKicks must not be set when latest open row is permanent")
	}
}

func TestLoadModerationStateLatestOpenRowWins(t *testing.T) {
	tdir := t.TempDir()
	db, err := InitDB(filepath.Join(tdir, "legacy.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer CloseDB(db)
	CreateSchema(db)

	kickExpiry := time.Now().Add(12 * time.Hour)
	// Legacy duplicate open rows: older permanent, newer kick (without close-before-insert).
	if _, err := dbExec(db, `INSERT INTO ban_history (username, banned_by, expires_at) VALUES (?, ?, ?)`, "legacy", "admin", nil); err != nil {
		t.Fatalf("insert permanent: %v", err)
	}
	if _, err := dbExec(db, `INSERT INTO ban_history (username, banned_by, expires_at) VALUES (?, ?, ?)`, "legacy", "admin", kickExpiry); err != nil {
		t.Fatalf("insert kick: %v", err)
	}

	hub := mustNewHub(t, tdir, tdir, "", db)
	hub.banMutex.RLock()
	_, permanent := hub.bans["legacy"]
	loadedExpiry, kicked := hub.tempKicks["legacy"]
	hub.banMutex.RUnlock()
	if permanent {
		t.Fatal("latest open row is temp kick; bans must be empty")
	}
	if !kicked || !time.Now().Before(loadedExpiry) {
		t.Fatalf("expected temp kick from latest open row, kicked=%v expiry=%v", kicked, loadedExpiry)
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
