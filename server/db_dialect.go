package server

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	mysqlerr "github.com/go-sql-driver/mysql"
)

type DBDialect string

const (
	DialectSQLite   DBDialect = "sqlite"
	DialectPostgres DBDialect = "postgres"
	DialectMySQL    DBDialect = "mysql"
)

// dbDialects stores a per-handle SQL dialect.
//
// Lifecycle note: entries are keyed by *sql.DB and are not explicitly deleted.
// Marchat keeps a single process-lifetime DB handle, so this map grows only if
// callers repeatedly create/close new handles in one process.
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

// touchUserLastSeenSQL records when a user last completed a successful handshake.
// last_message_id is legacy schema; replay no longer uses incremental catch-up.
func touchUserLastSeenSQL(db *sql.DB) string {
	switch getDBDialect(db) {
	case DialectPostgres:
		return `INSERT INTO user_message_state (username, last_message_id, last_seen) VALUES (?, 0, CURRENT_TIMESTAMP)
		ON CONFLICT (username) DO UPDATE SET last_seen = EXCLUDED.last_seen`
	case DialectMySQL:
		return `INSERT INTO user_message_state (username, last_message_id, last_seen) VALUES (?, 0, CURRENT_TIMESTAMP)
		ON DUPLICATE KEY UPDATE last_seen = CURRENT_TIMESTAMP`
	default:
		return `INSERT INTO user_message_state (username, last_message_id, last_seen) VALUES (?, 0, CURRENT_TIMESTAMP)
		ON CONFLICT(username) DO UPDATE SET last_seen = excluded.last_seen`
	}
}

// boolSQLLiteral returns a dialect-safe boolean literal for SQL (WHERE/SET/COALESCE).
// Postgres rejects boolean = integer; SQLite and MySQL accept 0/1.
func boolSQLLiteral(db *sql.DB, value bool) string {
	if getDBDialect(db) == DialectPostgres {
		if value {
			return "TRUE"
		}
		return "FALSE"
	}
	if value {
		return "1"
	}
	return "0"
}

func coalesceBoolSQL(db *sql.DB, column string) string {
	return fmt.Sprintf("COALESCE(%s, %s)", column, boolSQLLiteral(db, false))
}

// messageHistoryRowSelectColumns is the shared SELECT list for message history rows.
func messageHistoryRowSelectColumns(db *sql.DB) string {
	return fmt.Sprintf(
		"sender, content, created_at, is_encrypted, message_id, %s, %s, COALESCE(recipient, ''), COALESCE(channel, 'general')",
		coalesceBoolSQL(db, "edited"), coalesceBoolSQL(db, "deleted"),
	)
}

func searchMessagesSQL(db *sql.DB) string {
	return fmt.Sprintf(
		`SELECT sender, content, created_at, is_encrypted, message_id FROM messages WHERE content LIKE ? AND deleted = %s ORDER BY created_at DESC LIMIT ?`,
		boolSQLLiteral(db, false),
	)
}

func pinnedMessagesSQL(db *sql.DB) string {
	return fmt.Sprintf(
		`SELECT sender, content, created_at, message_id FROM messages WHERE pinned = %s ORDER BY created_at DESC`,
		boolSQLLiteral(db, true),
	)
}

func editMessageUpdateSQL(db *sql.DB) string {
	return fmt.Sprintf(
		`UPDATE messages SET content = ?, edited = %s, is_encrypted = ? WHERE message_id = ? AND sender = ?`,
		boolSQLLiteral(db, true),
	)
}

func deleteMessageUpdateSQL(db *sql.DB, requireSender bool) string {
	q := fmt.Sprintf(
		`UPDATE messages SET content = '[deleted]', deleted = %s, is_encrypted = %s WHERE message_id = ?`,
		boolSQLLiteral(db, true), boolSQLLiteral(db, false),
	)
	if requireSender {
		return q + ` AND sender = ?`
	}
	return q
}

// visibleMessagesForUserSQL returns up to limit rows visible to lowerUsername:
// channel/public rows (empty recipient) plus DMs where the user is sender or recipient.
func visibleMessagesForUserSQL(db *sql.DB) string {
	return fmt.Sprintf(`SELECT %s
FROM messages
WHERE (recipient IS NULL OR TRIM(recipient) = '' OR LOWER(TRIM(sender)) = ? OR LOWER(TRIM(recipient)) = ?)
ORDER BY created_at DESC
LIMIT ?`, messageHistoryRowSelectColumns(db))
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

// isMySQLDuplicateKeyName reports whether err is MySQL errno 1061 (index already exists).
func isMySQLDuplicateKeyName(err error) bool {
	var me *mysqlerr.MySQLError
	return errors.As(err, &me) && me.Number == 1061
}
