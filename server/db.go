package server

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		log.Fatal(err)
	}

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

	return db
}
