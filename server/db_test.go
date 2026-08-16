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
	if !strings.Contains(plain, "_txlock=immediate") {
		t.Fatalf("plain path missing _txlock=immediate: %q", plain)
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
	defer CloseDB(db)

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
	defer CloseDB(db)

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
		t.Fatalf("writer MaxOpenConnections = %d, want 1", stats.MaxOpenConnections)
	}

	read := dbRead(db)
	if read == nil || read == db {
		t.Fatal("file-backed InitDB should pair a distinct reader pool")
	}
	readStats := read.Stats()
	if readStats.MaxOpenConnections != sqliteReadMaxOpenConns {
		t.Fatalf("reader MaxOpenConnections = %d, want %d", readStats.MaxOpenConnections, sqliteReadMaxOpenConns)
	}
	var readBusy int
	if err := read.QueryRow("PRAGMA busy_timeout;").Scan(&readBusy); err != nil {
		t.Fatalf("reader PRAGMA busy_timeout: %v", err)
	}
	if readBusy <= 0 {
		t.Fatalf("reader busy_timeout = %d, want > 0", readBusy)
	}
	var queryOnly int
	if err := read.QueryRow("PRAGMA query_only;").Scan(&queryOnly); err != nil {
		t.Fatalf("PRAGMA query_only: %v", err)
	}
	if queryOnly == 0 {
		t.Fatal("reader query_only = 0, want on")
	}
}

func TestInitDBSQLiteMemory(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB(:memory:): %v", err)
	}
	defer CloseDB(db)

	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout;").Scan(&busyTimeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if busyTimeout <= 0 {
		t.Fatalf("busy_timeout = %d, want > 0", busyTimeout)
	}

	if dbRead(db) != db {
		t.Fatal("memory InitDB should not split a reader pool")
	}
	if db.Stats().MaxOpenConnections != 1 {
		t.Fatalf("memory MaxOpenConnections = %d, want 1", db.Stats().MaxOpenConnections)
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
	defer CloseDB(db)

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
	defer CloseDB(db)
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

	var readN int
	if err := dbRead(db).QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&readN); err != nil {
		t.Fatalf("reader count: %v", err)
	}
	if readN != n {
		t.Fatalf("reader message count = %d, want %d (writer inserts must be visible after commit)", readN, n)
	}
}

func TestSQLiteWALReadersProceedDuringWriteTx(t *testing.T) {
	tdir := t.TempDir()
	db, err := InitDB(filepath.Join(tdir, "walread.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer CloseDB(db)
	CreateSchema(db)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`INSERT INTO messages (sender, content, created_at, is_encrypted, recipient, channel) VALUES (?, ?, ?, ?, ?, ?)`,
		"user", "held-write", time.Now(), false, "", "general")
	if err != nil {
		t.Fatalf("insert in open tx: %v", err)
	}

	read := dbRead(db)
	if read == db {
		t.Fatal("expected distinct reader pool for file-backed SQLite")
	}

	done := make(chan error, 1)
	go func() {
		var n int
		done <- read.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&n)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("reader SELECT during open write tx: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reader SELECT blocked behind open write transaction")
	}
}

func TestInsertMessageVisibleOnReaderPoolAndReplay(t *testing.T) {
	tdir := t.TempDir()
	db, err := InitDB(filepath.Join(tdir, "persist.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer CloseDB(db)
	CreateSchema(db)

	const body = "persist-roundtrip-payload"
	id, err := InsertMessage(db, shared.Message{
		Sender:    "carol",
		Content:   body,
		CreatedAt: time.Now(),
		Channel:   "general",
	})
	if err != nil {
		t.Fatalf("InsertMessage: %v", err)
	}
	if id <= 0 {
		t.Fatalf("InsertMessage id = %d, want > 0", id)
	}

	var n int
	if err := dbQueryRow(dbRead(db), `SELECT COUNT(*) FROM messages WHERE content = ?`, body).Scan(&n); err != nil {
		t.Fatalf("reader lookup: %v", err)
	}
	if n != 1 {
		t.Fatalf("reader saw %d rows for inserted content, want 1", n)
	}

	recent := GetRecentMessages(db)
	found := false
	for _, msg := range recent {
		if msg.Content == body && msg.Sender == "carol" && msg.MessageID == id {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("GetRecentMessages missing persisted message id=%d; got %+v", id, recent)
	}
}
