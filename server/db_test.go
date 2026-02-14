package server

import (
	"path/filepath"
	"testing"
)

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
