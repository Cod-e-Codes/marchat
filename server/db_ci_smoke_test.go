package server

import (
	"database/sql"
	"os"
	"testing"
)

// CI sets MARCHAT_CI_POSTGRES_URL and MARCHAT_CI_MYSQL_URL when Postgres / MySQL (or MariaDB)
// service containers are available. Local runs skip unless those variables are set.

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
