package server

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
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

func TestMessageHistoryRowSelectColumnsPostgres(t *testing.T) {
	db := &sql.DB{}
	setDBDialect(db, DialectPostgres)
	got := messageHistoryRowSelectColumns(db)
	if !strings.Contains(got, "COALESCE(edited, FALSE)") {
		t.Fatalf("postgres edited coalesce: %s", got)
	}
	if strings.Contains(got, "COALESCE(edited, 0)") {
		t.Fatalf("postgres must not use integer coalesce for booleans: %s", got)
	}
}

func TestBoolSQLLiteralPostgres(t *testing.T) {
	db := &sql.DB{}
	setDBDialect(db, DialectPostgres)
	if got := boolSQLLiteral(db, true); got != "TRUE" {
		t.Fatalf("true literal: %q", got)
	}
	if got := boolSQLLiteral(db, false); got != "FALSE" {
		t.Fatalf("false literal: %q", got)
	}
}

func TestSearchMessagesSQLPostgres(t *testing.T) {
	db := &sql.DB{}
	setDBDialect(db, DialectPostgres)
	got := searchMessagesSQL(db)
	if !strings.Contains(got, "deleted = FALSE") {
		t.Fatalf("postgres search must use FALSE: %s", got)
	}
	if strings.Contains(got, "deleted = 0") {
		t.Fatalf("postgres search must not use integer boolean: %s", got)
	}
}

func TestPinnedMessagesSQLPostgres(t *testing.T) {
	db := &sql.DB{}
	setDBDialect(db, DialectPostgres)
	got := pinnedMessagesSQL(db)
	if !strings.Contains(got, "pinned = TRUE") {
		t.Fatalf("postgres pinned list must use TRUE: %s", got)
	}
}

func TestPrepareMySQLDSN(t *testing.T) {
	got, err := prepareMySQLDSN("user:pass@tcp(localhost:3306)/marchat")
	if err != nil {
		t.Fatalf("prepareMySQLDSN: %v", err)
	}
	if !strings.Contains(got, "parseTime=true") {
		t.Fatalf("expected parseTime=true, got %q", got)
	}

	got2, err := prepareMySQLDSN("user:pass@tcp(localhost:3306)/marchat?parseTime=false")
	if err != nil {
		t.Fatalf("prepareMySQLDSN parseTime=false: %v", err)
	}
	if !strings.Contains(got2, "parseTime=true") {
		t.Fatalf("parseTime=false must be overridden, got %q", got2)
	}
}

func TestPrepareMySQLDSNPreservesPassword(t *testing.T) {
	cfg := mysql.Config{
		User:   "user",
		Passwd: "p@ss:w&rd",
		Net:    "tcp",
		Addr:   "localhost:3306",
		DBName: "marchat",
	}
	got, err := prepareMySQLDSN(cfg.FormatDSN())
	if err != nil {
		t.Fatalf("prepareMySQLDSN: %v", err)
	}
	parsed, err := mysql.ParseDSN(got)
	if err != nil {
		t.Fatalf("ParseDSN round-trip: %v", err)
	}
	if parsed.Passwd != cfg.Passwd {
		t.Fatalf("password round-trip: got %q want %q", parsed.Passwd, cfg.Passwd)
	}
	if !parsed.ParseTime {
		t.Fatal("expected ParseTime true")
	}
}
