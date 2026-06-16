package server

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
)

// CI sets MARCHAT_CI_POSTGRES_URL and MARCHAT_CI_MYSQL_URL when Postgres / MySQL (or MariaDB)
// service containers are available. Local runs skip unless those variables are set.
// MySQL DSNs must start with mysql: or mysql:// (see detectDriver in db.go); otherwise InitDB uses SQLite.

func TestPostgresInitDBAndSchemaSmoke(t *testing.T) {
	dsn := os.Getenv("MARCHAT_CI_POSTGRES_URL")
	if dsn == "" {
		t.Skip("set MARCHAT_CI_POSTGRES_URL to run Postgres smoke (see .github/workflows/go.yml)")
	}
	db, err := InitDB(dsn)
	if err != nil {
		t.Fatalf("InitDB postgres: %v", err)
	}
	defer db.Close()

	if getDBDialect(db) != DialectPostgres {
		t.Fatalf("dialect = %v, want postgres", getDBDialect(db))
	}

	CreateSchema(db)
	assertCISmokeTables(t, db, "postgres")
	assertCIHandshakeReplaySmoke(t, db)
	assertCISearchAndPinSmoke(t, db)
}

func TestMySQLInitDBAndSchemaSmoke(t *testing.T) {
	dsn := os.Getenv("MARCHAT_CI_MYSQL_URL")
	if dsn == "" {
		t.Skip("set MARCHAT_CI_MYSQL_URL to run MySQL/MariaDB smoke (see .github/workflows/go.yml)")
	}
	db, err := InitDB(dsn)
	if err != nil {
		t.Fatalf("InitDB mysql: %v", err)
	}
	defer db.Close()

	if getDBDialect(db) != DialectMySQL {
		t.Fatalf("dialect = %v, want mysql", getDBDialect(db))
	}

	CreateSchema(db)
	assertCISmokeTables(t, db, "mysql")
	assertCIHandshakeReplaySmoke(t, db)
	assertCISearchAndPinSmoke(t, db)
}

func assertCISmokeTables(t *testing.T, db *sql.DB, kind string) {
	t.Helper()
	tables := []string{"messages", "user_message_state", "ban_history"}
	for _, name := range tables {
		var n int
		var err error
		switch kind {
		case "postgres":
			err = dbQueryRow(db, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = ?`, name).Scan(&n)
		case "mysql":
			err = dbQueryRow(db, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, name).Scan(&n)
		default:
			t.Fatalf("unknown kind %q", kind)
		}
		if err != nil {
			t.Fatalf("table %q lookup: %v", name, err)
		}
		if n != 1 {
			t.Fatalf("expected table %q to exist (count=%d)", name, n)
		}
	}
}

func assertCIHandshakeReplaySmoke(t *testing.T, db *sql.DB) {
	t.Helper()
	invalidateRecentMessagesCache()

	now := time.Now()
	if _, err := InsertMessage(db, shared.Message{
		Sender:    "alice",
		Content:   "ci-smoke-public",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("InsertMessage public: %v", err)
	}
	if _, err := InsertMessage(db, shared.Message{
		Sender:    "alice",
		Recipient: "carol",
		Type:      shared.DirectMessage,
		Content:   "ci-smoke-dm",
		CreatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("InsertMessage dm: %v", err)
	}

	bobMsgs := GetRecentMessagesForUser(db, "bob", HandshakeReplayLimit, false)
	if len(bobMsgs) != 1 {
		t.Fatalf("bob visible replay len = %d, want 1", len(bobMsgs))
	}
	if bobMsgs[0].Content != "ci-smoke-public" {
		t.Fatalf("bob visible replay content = %q, want ci-smoke-public", bobMsgs[0].Content)
	}

	carolMsgs := GetRecentMessagesForUser(db, "carol", HandshakeReplayLimit, false)
	if len(carolMsgs) != 2 {
		t.Fatalf("carol visible replay len = %d, want 2", len(carolMsgs))
	}
}

func assertCISearchAndPinSmoke(t *testing.T, db *sql.DB) {
	t.Helper()
	invalidateRecentMessagesCache()

	now := time.Now()
	activeID, err := InsertMessage(db, shared.Message{
		Sender:    "alice",
		Content:   "ci-smoke-searchable-term",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("InsertMessage active: %v", err)
	}
	delID, err := InsertMessage(db, shared.Message{
		Sender:    "bob",
		Content:   "ci-smoke-searchable-term",
		CreatedAt: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("InsertMessage deletable: %v", err)
	}
	if err := DeleteMessage(db, delID, "bob", false); err != nil {
		t.Fatalf("DeleteMessage: %v", err)
	}

	found := SearchMessages(db, "ci-smoke-searchable-term", 10)
	if len(found) != 1 {
		t.Fatalf("search len = %d, want 1", len(found))
	}
	if found[0].MessageID != activeID {
		t.Fatalf("search message_id = %d, want %d", found[0].MessageID, activeID)
	}

	pinned, err := TogglePinMessage(db, activeID)
	if err != nil {
		t.Fatalf("TogglePinMessage: %v", err)
	}
	if !pinned {
		t.Fatal("expected pinned true after toggle")
	}
	pinnedList := GetPinnedMessages(db)
	if len(pinnedList) != 1 {
		t.Fatalf("pinned list len = %d, want 1", len(pinnedList))
	}
	if pinnedList[0].MessageID != activeID {
		t.Fatalf("pinned message_id = %d, want %d", pinnedList[0].MessageID, activeID)
	}
}
