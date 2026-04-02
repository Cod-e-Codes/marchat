package server

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
)

type DBDialect string

const (
	DialectSQLite   DBDialect = "sqlite"
	DialectPostgres DBDialect = "postgres"
	DialectMySQL    DBDialect = "mysql"
)

var dbDialects sync.Map // map[*sql.DB]DBDialect

func setDBDialect(db *sql.DB, dialect DBDialect) {
	if db != nil {
		dbDialects.Store(db, dialect)
	}
}

func getDBDialect(db *sql.DB) DBDialect {
	if db == nil {
		return DialectSQLite
	}
	if v, ok := dbDialects.Load(db); ok {
		if d, ok2 := v.(DBDialect); ok2 {
			return d
		}
	}
	return DialectSQLite
}

func rebindQuery(db *sql.DB, query string) string {
	if getDBDialect(db) != DialectPostgres {
		return query
	}
	var b strings.Builder
	idx := 1
	for _, ch := range query {
		if ch == '?' {
			b.WriteString(fmt.Sprintf("$%d", idx))
			idx++
			continue
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func dbExec(db *sql.DB, query string, args ...interface{}) (sql.Result, error) {
	return db.Exec(rebindQuery(db, query), args...)
}

func dbQuery(db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
	return db.Query(rebindQuery(db, query), args...)
}

func dbQueryRow(db *sql.DB, query string, args ...interface{}) *sql.Row {
	return db.QueryRow(rebindQuery(db, query), args...)
}

func upsertUserMessageStateSQL(db *sql.DB) string {
	switch getDBDialect(db) {
	case DialectPostgres:
		return `INSERT INTO user_message_state (username, last_message_id, last_seen) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (username) DO UPDATE SET last_message_id = EXCLUDED.last_message_id, last_seen = EXCLUDED.last_seen`
	case DialectMySQL:
		return `INSERT INTO user_message_state (username, last_message_id, last_seen) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON DUPLICATE KEY UPDATE last_message_id = VALUES(last_message_id), last_seen = VALUES(last_seen)`
	default:
		return `INSERT OR REPLACE INTO user_message_state (username, last_message_id, last_seen) VALUES (?, ?, CURRENT_TIMESTAMP)`
	}
}

func upsertUserChannelSQL(db *sql.DB) string {
	switch getDBDialect(db) {
	case DialectPostgres:
		return `INSERT INTO user_channels (username, channel, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (username) DO UPDATE SET channel = EXCLUDED.channel, updated_at = EXCLUDED.updated_at`
	case DialectMySQL:
		return `INSERT INTO user_channels (username, channel, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON DUPLICATE KEY UPDATE channel = VALUES(channel), updated_at = VALUES(updated_at)`
	default:
		return `INSERT OR REPLACE INTO user_channels (username, channel, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)`
	}
}

func insertIgnoreReactionSQL(db *sql.DB) string {
	switch getDBDialect(db) {
	case DialectPostgres:
		return `INSERT INTO message_reactions (message_id, username, emoji) VALUES (?, ?, ?) ON CONFLICT (message_id, username, emoji) DO NOTHING`
	case DialectMySQL:
		return `INSERT IGNORE INTO message_reactions (message_id, username, emoji) VALUES (?, ?, ?)`
	default:
		return `INSERT OR IGNORE INTO message_reactions (message_id, username, emoji) VALUES (?, ?, ?)`
	}
}

func insertIgnoreReadReceiptSQL(db *sql.DB) string {
	switch getDBDialect(db) {
	case DialectPostgres:
		return `INSERT INTO read_receipts (username, message_id, read_at) VALUES (?, ?, CURRENT_TIMESTAMP) ON CONFLICT (username, message_id) DO NOTHING`
	case DialectMySQL:
		return `INSERT IGNORE INTO read_receipts (username, message_id, read_at) VALUES (?, ?, CURRENT_TIMESTAMP)`
	default:
		return `INSERT OR IGNORE INTO read_receipts (username, message_id, read_at) VALUES (?, ?, CURRENT_TIMESTAMP)`
	}
}

func supportsLastInsertID(db *sql.DB) bool {
	return getDBDialect(db) != DialectPostgres
}
