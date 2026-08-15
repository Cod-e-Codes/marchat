package server

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateSchemaFromEmpty(t *testing.T) {
	t.Run("memory", func(t *testing.T) {
		db, err := InitDB(":memory:")
		if err != nil {
			t.Fatalf("InitDB: %v", err)
		}
		defer db.Close()
		assertMigrateSchemaFromEmpty(t, db)
	})

	t.Run("file", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "migrate.db")
		db, err := InitDB(dbPath)
		if err != nil {
			t.Fatalf("InitDB: %v", err)
		}
		defer db.Close()
		assertMigrateSchemaFromEmpty(t, db)
	})
}

func assertMigrateSchemaFromEmpty(t *testing.T, db *sql.DB) {
	t.Helper()

	if err := MigrateSchema(db); err != nil {
		t.Fatalf("first MigrateSchema: %v", err)
	}

	version, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if version != 1 {
		t.Fatalf("schema version = %d, want 1", version)
	}

	for _, table := range []string{
		"messages", "user_message_state", "ban_history", "message_reactions",
		"user_channels", "read_receipts", "schema_version",
	} {
		ok, err := tableExists(db, table)
		if err != nil {
			t.Fatalf("tableExists(%q): %v", table, err)
		}
		if !ok {
			t.Fatalf("table %q missing after migration", table)
		}
	}

	hasExpires, err := columnExists(db, "ban_history", "expires_at")
	if err != nil {
		t.Fatalf("columnExists ban_history.expires_at: %v", err)
	}
	if !hasExpires {
		t.Fatal("ban_history.expires_at column missing")
	}

	if err := MigrateSchema(db); err != nil {
		t.Fatalf("second MigrateSchema (idempotent): %v", err)
	}
}

func TestMigrateSchemaFailsWhenRecordedSchemaIsIncomplete(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer db.Close()

	if err := ensureSchemaVersionTable(db); err != nil {
		t.Fatalf("ensureSchemaVersionTable: %v", err)
	}
	if _, err := dbExec(db, `INSERT INTO schema_version (version) VALUES (1)`); err != nil {
		t.Fatalf("seed schema_version: %v", err)
	}

	// Version claims v1 but required table is missing (corruption / incomplete install).
	if err := applyMigrationV1(migrationConn{db: db}); err != nil {
		t.Fatalf("applyMigrationV1: %v", err)
	}
	if _, err := dbExec(db, `DROP TABLE ban_history`); err != nil {
		t.Fatalf("drop ban_history: %v", err)
	}

	err = MigrateSchema(db)
	if err == nil {
		t.Fatal("expected MigrateSchema to fail when required tables are missing")
	}
	if !strings.Contains(err.Error(), "ban_history") {
		t.Fatalf("error = %v, want mention of missing ban_history", err)
	}
}

func TestMigrateSchemaV1RollsBackOnInjectedFailure(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer db.Close()

	migrationFailAfterStep = "create user_message_state"
	t.Cleanup(func() { migrationFailAfterStep = "" })

	err = MigrateSchema(db)
	if err == nil {
		t.Fatal("expected MigrateSchema to fail on injected mid-migration error")
	}
	if !strings.Contains(err.Error(), "injected migration failure") {
		t.Fatalf("error = %v, want injected failure", err)
	}

	version, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if version != 0 {
		t.Fatalf("schema version = %d after failed migration, want 0", version)
	}

	// messages + user_message_state were created inside the rolled-back transaction.
	for _, table := range []string{"messages", "user_message_state", "ban_history"} {
		ok, err := tableExists(db, table)
		if err != nil {
			t.Fatalf("tableExists(%q): %v", table, err)
		}
		if ok {
			t.Fatalf("table %q should not exist after migration rollback", table)
		}
	}

	migrationFailAfterStep = ""
	if err := MigrateSchema(db); err != nil {
		t.Fatalf("retry MigrateSchema after rollback: %v", err)
	}
	version, err = readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion after retry: %v", err)
	}
	if version != 1 {
		t.Fatalf("schema version = %d after retry, want 1", version)
	}
}
