package server

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
)

func TestAppendSQLiteDSNPragmas(t *testing.T) {
	t.Parallel()

	plain := appendSQLiteDSNPragmas("/tmp/chat.db")
	if !strings.HasPrefix(plain, "/tmp/chat.db?") {
		t.Fatalf("plain path DSN = %q, want ?-joined query", plain)
	}
	if !strings.Contains(plain, "_busy_timeout=5000") {
		t.Fatalf("plain path missing busy_timeout: %q", plain)
	}

	withQuery := appendSQLiteDSNPragmas("file:test.db?mode=rwc")
	if !strings.HasPrefix(withQuery, "file:test.db?mode=rwc&") {
		t.Fatalf("query path DSN = %q, want &-joined pragmas", withQuery)
	}
	if strings.Contains(withQuery, "?mode=rwc?") {
		t.Fatalf("double ? in DSN: %q", withQuery)
	}
}

func TestInitDBAndSchema(t *testing.T) {
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatalf("db is nil")
		return
	}

	CreateSchema(db)
	// basic smoke: query created tables exist
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='messages'").Scan(&n); err != nil {
		t.Fatalf("query messages table: %v", err)
	}
	if n == 0 {
		t.Fatalf("messages table not created")
	}

	// user_message_state should exist
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='user_message_state'").Scan(&n); err != nil {
		t.Fatalf("query user_message_state: %v", err)
	}
	if n == 0 {
		t.Fatalf("user_message_state table not created")
	}
}

func TestInitDBSQLiteFilePragmasAndPool(t *testing.T) {
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "pragmas.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer db.Close()

	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout;").Scan(&busyTimeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if busyTimeout <= 0 {
		t.Fatalf("busy_timeout = %d, want > 0", busyTimeout)
	}

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	stats := db.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1", stats.MaxOpenConnections)
	}
}

func TestInitDBSQLiteMemory(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB(:memory:): %v", err)
	}
	defer db.Close()

	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout;").Scan(&busyTimeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if busyTimeout <= 0 {
		t.Fatalf("busy_timeout = %d, want > 0", busyTimeout)
	}

	CreateSchema(db)
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='messages'").Scan(&n); err != nil {
		t.Fatalf("messages table: %v", err)
	}
	if n == 0 {
		t.Fatal("messages table not created")
	}
}

func TestInitDBSQLitePathWithExistingQuery(t *testing.T) {
	tdir := t.TempDir()
	// modernc accepts file: URIs; mode=rwc creates the file if missing.
	path := filepath.ToSlash(filepath.Join(tdir, "query.db"))
	dsn := "file:" + path + "?mode=rwc"
	db, err := InitDB(dsn)
	if err != nil {
		t.Fatalf("InitDB(%q): %v", dsn, err)
	}
	defer db.Close()

	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout;").Scan(&busyTimeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if busyTimeout <= 0 {
		t.Fatalf("busy_timeout = %d, want > 0", busyTimeout)
	}

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}
}

func TestIsSQLiteMemoryDSN(t *testing.T) {
	t.Parallel()
	cases := []struct {
		dsn  string
		want bool
	}{
		{":memory:", true},
		{":memory:?_busy_timeout=5000", true},
		{"file::memory:", true},
		{"file:memdb?mode=memory", true},
		{"/tmp/chat.db", false},
		{"file:/tmp/chat.db?mode=rwc", false},
	}
	for _, tc := range cases {
		if got := isSQLiteMemoryDSN(tc.dsn); got != tc.want {
			t.Errorf("isSQLiteMemoryDSN(%q) = %v, want %v", tc.dsn, got, tc.want)
		}
	}
}

func TestInitDBSQLiteConcurrentInserts(t *testing.T) {
	tdir := t.TempDir()
	db, err := InitDB(filepath.Join(tdir, "contention.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer db.Close()
	CreateSchema(db)

	const goroutines = 8
	const perG = 20
	var wg sync.WaitGroup
	errCh := make(chan error, goroutines*perG)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				_, err := InsertMessage(db, shared.Message{
					Sender:    "user",
					Content:   "msg",
					CreatedAt: time.Now(),
					Channel:   "general",
				})
				if err != nil {
					errCh <- err
				}
			}
		}(g)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent insert: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != goroutines*perG {
		t.Fatalf("message count = %d, want %d", n, goroutines*perG)
	}
}
