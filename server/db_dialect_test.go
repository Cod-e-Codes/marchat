package server

import (
	"database/sql"
	"testing"
)

func TestDetectDriver(t *testing.T) {
	cases := []struct {
		in      string
		driver  string
		dialect DBDialect
	}{
		{"/tmp/marchat.db", "sqlite", DialectSQLite},
		{"postgres://user:pass@localhost:5432/marchat?sslmode=disable", "pgx", DialectPostgres},
		{"mysql:user:pass@tcp(localhost:3306)/marchat", "mysql", DialectMySQL},
		{"mysql://user:pass@tcp(localhost:3306)/marchat", "mysql", DialectMySQL},
	}

	for _, tc := range cases {
		driver, dialect, _ := detectDriver(tc.in)
		if driver != tc.driver || dialect != tc.dialect {
			t.Fatalf("detectDriver(%q) = (%s,%s), want (%s,%s)", tc.in, driver, dialect, tc.driver, tc.dialect)
		}
	}
}

func TestRebindQueryPostgres(t *testing.T) {
	db := &sql.DB{}
	setDBDialect(db, DialectPostgres)
	got := rebindQuery(db, "SELECT * FROM t WHERE a = ? AND b = ?")
	if got != "SELECT * FROM t WHERE a = $1 AND b = $2" {
		t.Fatalf("unexpected rebind output: %s", got)
	}
}
