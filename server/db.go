package server

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

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

func InitDB(conn string) (*sql.DB, error) {
	driver, dialect, dsn := detectDriver(conn)

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
		// Enable WAL mode for better concurrency and performance
		_, err = db.Exec("PRAGMA journal_mode=WAL;")
		if err != nil {
			log.Printf("Warning: Could not enable WAL mode: %v", err)
		} else {
			// Verify WAL mode was actually enabled
			var journalMode string
			err = db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode)
			if err != nil {
				log.Printf("Warning: Could not verify journal mode: %v", err)
			} else {
				log.Printf("Database journal mode set to %s for improved concurrency", journalMode)
			}
		}

		// Set additional performance optimizations
		_, err = db.Exec("PRAGMA synchronous=NORMAL;")
		if err != nil {
			log.Printf("Warning: Could not set synchronous mode: %v", err)
		}

		_, err = db.Exec("PRAGMA cache_size=10000;")
		if err != nil {
			log.Printf("Warning: Could not set cache size: %v", err)
		}

		_, err = db.Exec("PRAGMA temp_store=MEMORY;")
		if err != nil {
			log.Printf("Warning: Could not set temp store: %v", err)
		}
	}

	return db, nil
}
