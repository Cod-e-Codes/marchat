package server

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// sqliteConnPragmas are applied on every new SQLite connection via the DSN
// (modernc.org/sqlite v1.55.0 shorthands + _pragma). One-shot PRAGMA Exec after
// Open only affects the connection that ran it, so these must live in the DSN.
const sqliteConnPragmas = "_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_pragma=cache_size(10000)&_pragma=temp_store(MEMORY)"

func detectDriver(conn string) (string, DBDialect, string) {
	v := strings.TrimSpace(conn)
	switch {
	case strings.HasPrefix(v, "postgres://"), strings.HasPrefix(v, "postgresql://"):
		return "pgx", DialectPostgres, v
	case strings.HasPrefix(v, "mysql://"):
		return "mysql", DialectMySQL, strings.TrimPrefix(v, "mysql://")
	case strings.HasPrefix(v, "mysql:"):
		return "mysql", DialectMySQL, strings.TrimPrefix(v, "mysql:")
	default:
		return "sqlite", DialectSQLite, v
	}
}

func prepareMySQLDSN(dsn string) (string, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", fmt.Errorf("parse mysql dsn: %w", err)
	}
	if !cfg.ParseTime {
		log.Printf("Warning: MySQL DSN has parseTime=false; marchat requires parseTime=true for DATETIME scans")
	}
	cfg.ParseTime = true
	if cfg.Loc == nil {
		cfg.Loc = time.Local
	}
	return cfg.FormatDSN(), nil
}

// appendSQLiteDSNPragmas joins per-connection PRAGMA query params onto a SQLite
// path or DSN, using & when a query string is already present.
func appendSQLiteDSNPragmas(path string) string {
	if strings.Contains(path, "?") {
		return path + "&" + sqliteConnPragmas
	}
	return path + "?" + sqliteConnPragmas
}

// isSQLiteMemoryDSN reports whether the SQLite DSN is an in-memory database
// where WAL may not stick (PRAGMA journal_mode often remains "memory").
func isSQLiteMemoryDSN(dsn string) bool {
	base := dsn
	if i := strings.Index(dsn, "?"); i >= 0 {
		base = dsn[:i]
	}
	base = strings.TrimSpace(base)
	if base == ":memory:" || strings.EqualFold(base, "file::memory:") {
		return true
	}
	q := ""
	if i := strings.Index(dsn, "?"); i >= 0 {
		q = strings.ToLower(dsn[i+1:])
	}
	return strings.Contains(q, "mode=memory")
}

func verifySQLitePragmas(db *sql.DB, dsn string) error {
	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout;").Scan(&busyTimeout); err != nil {
		return fmt.Errorf("verify PRAGMA busy_timeout: %w", err)
	}
	if busyTimeout <= 0 {
		return fmt.Errorf("PRAGMA busy_timeout = %d, want > 0", busyTimeout)
	}

	if isSQLiteMemoryDSN(dsn) {
		return nil
	}

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		return fmt.Errorf("verify PRAGMA journal_mode: %w", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		return fmt.Errorf("PRAGMA journal_mode = %q, want wal", journalMode)
	}
	return nil
}

func InitDB(conn string) (*sql.DB, error) {
	driver, dialect, dsn := detectDriver(conn)
	if dialect == DialectMySQL {
		var err error
		dsn, err = prepareMySQLDSN(dsn)
		if err != nil {
			return nil, err
		}
	}

	if dialect == DialectSQLite {
		dsn = appendSQLiteDSNPragmas(dsn)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s database: %w", dialect, err)
	}

	// Verify the connection works
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to %s database: %w", dialect, err)
	}

	setDBDialect(db, dialect)

	if dialect == DialectSQLite {
		// Single connection: SQLite does not benefit from a multi-conn pool and
		// concurrent writers on separate connections race for the write lock.
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		if err := verifySQLitePragmas(db, dsn); err != nil {
			db.Close()
			return nil, fmt.Errorf("sqlite pragma verification failed: %w", err)
		}
		log.Printf("SQLite connected with per-connection pragmas (busy_timeout, WAL for file DBs) and MaxOpenConns=1")
	}

	return db, nil
}
