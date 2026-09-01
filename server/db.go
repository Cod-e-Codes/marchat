package server

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// sqliteConnPragmas are applied on every new SQLite writer connection via the DSN
// (modernc.org/sqlite v1.55.0 shorthands + _pragma). One-shot PRAGMA Exec after
// Open only affects the connection that ran it, so these must live in the DSN.
// _txlock=immediate applies to Begin on the writer pool (MigrateSchema).
const sqliteConnPragmas = "_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_pragma=cache_size(10000)&_pragma=temp_store(MEMORY)&_txlock=immediate"

// sqliteReadPragmas are for the file-backed reader pool. WAL is already persistent
// on the database file after the writer Ping; do not set journal_mode or mode=ro
// here (mode=ro cannot create -wal/-shm). Do not use cache=shared.
const sqliteReadPragmas = "_busy_timeout=5000&_query_only=1"

const sqliteReadMaxOpenConns = 4

// sqliteReadPools maps the InitDB write *sql.DB to its sibling reader pool.
// File-backed SQLite only; memory / Postgres / MySQL have no entry.
var sqliteReadPools sync.Map // map[*sql.DB]*sql.DB

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

func joinSQLiteDSN(path, extra string) string {
	if strings.Contains(path, "?") {
		return path + "&" + extra
	}
	return path + "?" + extra
}

// appendSQLiteDSNPragmas joins per-connection writer PRAGMA query params onto a
// SQLite path or DSN, using & when a query string is already present.
func appendSQLiteDSNPragmas(path string) string {
	return joinSQLiteDSN(path, sqliteConnPragmas)
}

// isSQLiteMemoryDSN reports whether the SQLite DSN is an in-memory database
// where WAL may not stick (PRAGMA journal_mode often remains "memory") and a
// second connection would be a different database.
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

func openSQLiteReadPool(path string, dialect DBDialect) (*sql.DB, error) {
	dsn := joinSQLiteDSN(path, sqliteReadPragmas)
	read, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite reader: %w", err)
	}
	if err := read.Ping(); err != nil {
		_ = read.Close()
		return nil, fmt.Errorf("ping sqlite reader: %w", err)
	}
	read.SetMaxOpenConns(sqliteReadMaxOpenConns)
	read.SetMaxIdleConns(sqliteReadMaxOpenConns)
	setDBDialect(read, dialect)
	return read, nil
}

// dbRead returns the SQLite reader pool paired with db, or db itself when there
// is no sibling (memory SQLite, Postgres, MySQL, or a handle not from InitDB).
func dbRead(db *sql.DB) *sql.DB {
	if db == nil {
		return nil
	}
	if v, ok := sqliteReadPools.Load(db); ok {
		if r, ok := v.(*sql.DB); ok && r != nil {
			return r
		}
	}
	return db
}

// CloseDB closes the InitDB write handle and its sibling SQLite reader, if any.
func CloseDB(db *sql.DB) error {
	if db == nil {
		return nil
	}
	var readErr error
	if v, ok := sqliteReadPools.LoadAndDelete(db); ok {
		if r, ok := v.(*sql.DB); ok && r != nil && r != db {
			readErr = r.Close()
			dbDialects.Delete(r)
		}
	}
	dbDialects.Delete(db)
	err := db.Close()
	if err != nil && strings.Contains(err.Error(), "database is closed") {
		err = nil
	}
	if readErr != nil && strings.Contains(readErr.Error(), "database is closed") {
		readErr = nil
	}
	if readErr != nil {
		return readErr
	}
	return err
}

func InitDB(conn string) (*sql.DB, error) {
	driver, dialect, dsn := detectDriver(conn)
	origDSN := dsn
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
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to %s database: %w", dialect, err)
	}

	setDBDialect(db, dialect)

	if dialect == DialectSQLite {
		// Writer pool: SQLite allows one writer at a time. Concurrent writers on
		// separate connections race for the write lock (#118 SQLITE_BUSY).
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		if err := verifySQLitePragmas(db, origDSN); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("sqlite pragma verification failed: %w", err)
		}

		if !isSQLiteMemoryDSN(origDSN) {
			read, err := openSQLiteReadPool(origDSN, dialect)
			if err != nil {
				_ = db.Close()
				return nil, fmt.Errorf("sqlite reader pool: %w", err)
			}
			sqliteReadPools.Store(db, read)
			log.Printf("SQLite connected with per-connection pragmas (busy_timeout, WAL for file DBs); writer MaxOpenConns=1, reader MaxOpenConns=%d", sqliteReadMaxOpenConns)
		} else {
			log.Printf("SQLite in-memory: single pool MaxOpenConns=1 (no WAL reader split)")
		}
	}

	return db, nil
}
