package server

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

const currentSchemaVersion = 1

// migrationFailAfterStep is a test-only hook. When non-empty, applyMigrationV1
// returns an error after completing the named step (used to verify rollback).
var migrationFailAfterStep string

type schemaTypes struct {
	idColumn          string
	boolDefault       string
	boolMigrationDef  string
	dateTimeType      string
	blobType          string
	textType          string
	keyedTextType     string
	channelColumnType string
	banHistoryID      string
}

func schemaTypesForDialect(dialect DBDialect) schemaTypes {
	st := schemaTypes{
		idColumn:          "id INTEGER PRIMARY KEY AUTOINCREMENT",
		boolDefault:       "BOOLEAN DEFAULT 0",
		boolMigrationDef:  "BOOLEAN DEFAULT 0",
		dateTimeType:      "DATETIME",
		blobType:          "BLOB",
		textType:          "TEXT",
		keyedTextType:     "TEXT",
		channelColumnType: "TEXT",
		banHistoryID:      "id INTEGER PRIMARY KEY AUTOINCREMENT",
	}
	switch dialect {
	case DialectPostgres:
		st.idColumn = "id BIGSERIAL PRIMARY KEY"
		st.boolDefault = "BOOLEAN DEFAULT FALSE"
		st.boolMigrationDef = "BOOLEAN DEFAULT FALSE"
		st.dateTimeType = "TIMESTAMPTZ"
		st.blobType = "BYTEA"
		st.banHistoryID = "id BIGSERIAL PRIMARY KEY"
	case DialectMySQL:
		st.idColumn = "id BIGINT PRIMARY KEY AUTO_INCREMENT"
		st.boolDefault = "BOOLEAN DEFAULT FALSE"
		st.boolMigrationDef = "BOOLEAN DEFAULT FALSE"
		st.dateTimeType = "DATETIME"
		st.blobType = "LONGBLOB"
		st.textType = "LONGTEXT"
		st.keyedTextType = "VARCHAR(191)"
		st.channelColumnType = st.keyedTextType
		st.banHistoryID = "id BIGINT PRIMARY KEY AUTO_INCREMENT"
	}
	return st
}

// migrationConn executes SQL against either *sql.DB or *sql.Tx while keeping
// dialect rebinding keyed off the parent *sql.DB handle.
type migrationConn struct {
	db *sql.DB
	tx *sql.Tx
}

func (c migrationConn) Exec(query string, args ...interface{}) (sql.Result, error) {
	q := rebindQuery(c.db, query)
	if c.tx != nil {
		return c.tx.Exec(q, args...)
	}
	return c.db.Exec(q, args...)
}

func (c migrationConn) QueryRow(query string, args ...interface{}) *sql.Row {
	q := rebindQuery(c.db, query)
	if c.tx != nil {
		return c.tx.QueryRow(q, args...)
	}
	return c.db.QueryRow(q, args...)
}

func (c migrationConn) maybeFail(step string) error {
	if migrationFailAfterStep != "" && migrationFailAfterStep == step {
		return fmt.Errorf("injected migration failure after %s", step)
	}
	return nil
}

// MigrateSchema applies ordered schema migrations and verifies required tables exist.
// Existing databases without a schema_version row run the v1 baseline idempotently, then record version 1.
//
// SQLite and PostgreSQL apply each versioned migration (DDL + version row) inside a single
// transaction so a mid-migration failure rolls back. MySQL DDL implicitly commits, so MySQL
// runs steps without a multi-statement transaction; each statement is still statement-atomic
// on InnoDB, and schema_version is recorded only after applyMigrationV1 returns nil.
func MigrateSchema(db *sql.DB) error {
	if err := ensureSchemaVersionTable(db); err != nil {
		return fmt.Errorf("schema_version table: %w", err)
	}

	version, err := readSchemaVersion(db)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	if version < 1 {
		if err := applyVersionedMigration(db, 1, applyMigrationV1); err != nil {
			return fmt.Errorf("migration v1: %w", err)
		}
	}

	return verifySchema(db)
}

func applyVersionedMigration(db *sql.DB, version int, apply func(migrationConn) error) error {
	dialect := getDBDialect(db)
	if dialect == DialectMySQL {
		// MySQL DDL ends any open transaction (implicit commit). Do not wrap.
		if err := apply(migrationConn{db: db}); err != nil {
			return err
		}
		return setSchemaVersion(db, nil, version)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	conn := migrationConn{db: db, tx: tx}
	if err := apply(conn); err != nil {
		return err
	}
	if err := setSchemaVersion(db, tx, version); err != nil {
		return fmt.Errorf("record schema version %d: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration transaction: %w", err)
	}
	return nil
}

func ensureSchemaVersionTable(db *sql.DB) error {
	st := schemaTypesForDialect(getDBDialect(db))
	_, err := dbExec(db, fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at %s NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`, st.dateTimeType))
	return err
}

func readSchemaVersion(db *sql.DB) (int, error) {
	var version sql.NullInt64
	err := dbQueryRow(db, `SELECT MAX(version) FROM schema_version`).Scan(&version)
	if err != nil {
		return 0, err
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}

func setSchemaVersion(db *sql.DB, tx *sql.Tx, version int) error {
	conn := migrationConn{db: db, tx: tx}
	var q string
	switch getDBDialect(db) {
	case DialectPostgres:
		q = `INSERT INTO schema_version (version) VALUES (?) ON CONFLICT (version) DO NOTHING`
	case DialectMySQL:
		q = `INSERT IGNORE INTO schema_version (version) VALUES (?)`
	default:
		q = `INSERT OR IGNORE INTO schema_version (version) VALUES (?)`
	}
	_, err := conn.Exec(q, version)
	return err
}

func applyMigrationV1(conn migrationConn) error {
	dialect := getDBDialect(conn.db)
	st := schemaTypesForDialect(dialect)

	basicSchema := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS messages (
		%s,
		sender %s,
		content %s,
		created_at %s,
		is_encrypted %s,
		message_id INTEGER NOT NULL DEFAULT 0,
		edited %s,
		deleted %s,
		pinned %s,
		encrypted_data %s,
		nonce %s,
		recipient %s,
		channel %s NOT NULL DEFAULT 'general'
	);`, st.idColumn, st.textType, st.textType, st.dateTimeType, st.boolDefault,
		st.boolDefault, st.boolDefault, st.boolDefault, st.blobType, st.blobType, st.textType, st.channelColumnType)

	if _, err := conn.Exec(basicSchema); err != nil {
		return fmt.Errorf("create messages table: %w", err)
	}
	if err := conn.maybeFail("create messages"); err != nil {
		return err
	}

	migrations := []struct {
		column string
		ddl    string
	}{
		{"message_id", `ALTER TABLE messages ADD COLUMN message_id INTEGER DEFAULT 0`},
		{"edited", fmt.Sprintf("ALTER TABLE messages ADD COLUMN edited %s", st.boolMigrationDef)},
		{"deleted", fmt.Sprintf("ALTER TABLE messages ADD COLUMN deleted %s", st.boolMigrationDef)},
		{"pinned", fmt.Sprintf("ALTER TABLE messages ADD COLUMN pinned %s", st.boolMigrationDef)},
		{"channel", fmt.Sprintf("ALTER TABLE messages ADD COLUMN channel %s NOT NULL DEFAULT 'general'", st.channelColumnType)},
	}

	for _, m := range migrations {
		exists, err := columnExistsConn(conn, "messages", m.column)
		if err != nil {
			return fmt.Errorf("check messages.%s column: %w", m.column, err)
		}
		if !exists {
			if _, err := conn.Exec(m.ddl); err != nil {
				return fmt.Errorf("add messages.%s column: %w", m.column, err)
			}
		}
	}

	userStateSchema := `
	CREATE TABLE IF NOT EXISTS user_message_state (
		username ` + st.keyedTextType + ` PRIMARY KEY,
		last_message_id INTEGER NOT NULL DEFAULT 0,
		last_seen ` + st.dateTimeType + ` NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := conn.Exec(userStateSchema); err != nil {
		return fmt.Errorf("create user_message_state table: %w", err)
	}
	if err := conn.maybeFail("create user_message_state"); err != nil {
		return err
	}

	banHistorySchema := `
	CREATE TABLE IF NOT EXISTS ban_history (
		` + st.banHistoryID + `,
		username ` + st.keyedTextType + ` NOT NULL,
		banned_at ` + st.dateTimeType + ` NOT NULL DEFAULT CURRENT_TIMESTAMP,
		unbanned_at ` + st.dateTimeType + `,
		banned_by ` + st.keyedTextType + ` NOT NULL,
		expires_at ` + st.dateTimeType + `
	);`
	if _, err := conn.Exec(banHistorySchema); err != nil {
		return fmt.Errorf("create ban_history table: %w", err)
	}

	expiresExists, err := columnExistsConn(conn, "ban_history", "expires_at")
	if err != nil {
		return fmt.Errorf("check ban_history.expires_at column: %w", err)
	}
	if !expiresExists {
		if _, err := conn.Exec(`ALTER TABLE ban_history ADD COLUMN expires_at ` + st.dateTimeType); err != nil {
			return fmt.Errorf("add ban_history.expires_at column: %w", err)
		}
	}

	recipientIdx := `CREATE INDEX IF NOT EXISTS idx_messages_recipient ON messages(recipient)`
	if dialect == DialectMySQL {
		recipientIdx = `CREATE INDEX IF NOT EXISTS idx_messages_recipient ON messages(recipient(191))`
	}
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_messages_message_id ON messages(message_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at)`,
		recipientIdx,
		`CREATE INDEX IF NOT EXISTS idx_messages_deleted_created_at ON messages(deleted, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_user_message_state_username ON user_message_state(username)`,
		`CREATE INDEX IF NOT EXISTS idx_ban_history_username ON ban_history(username)`,
		`CREATE INDEX IF NOT EXISTS idx_ban_history_banned_at ON ban_history(banned_at)`,
		`CREATE INDEX IF NOT EXISTS idx_ban_history_unbanned_at ON ban_history(unbanned_at)`,
	}

	for _, index := range indexes {
		q := index
		if dialect == DialectMySQL {
			q = strings.Replace(index, "IF NOT EXISTS ", "", 1)
		}
		if _, err := conn.Exec(q); err != nil {
			if dialect == DialectMySQL && isMySQLDuplicateKeyName(err) {
				continue
			}
			return fmt.Errorf("create index %q: %w", index, err)
		}
	}

	if _, err := conn.Exec(`UPDATE messages SET message_id = id WHERE message_id = 0 OR message_id IS NULL`); err != nil {
		return fmt.Errorf("backfill messages.message_id: %w", err)
	}

	if _, err := conn.Exec(`
	CREATE TABLE IF NOT EXISTS message_reactions (
		` + st.idColumn + `,
		message_id INTEGER NOT NULL,
		username ` + st.keyedTextType + ` NOT NULL,
		emoji ` + st.keyedTextType + ` NOT NULL,
		created_at ` + st.dateTimeType + ` NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(message_id, username, emoji)
	);`); err != nil {
		return fmt.Errorf("create message_reactions table: %w", err)
	}

	if _, err := conn.Exec(`
	CREATE TABLE IF NOT EXISTS user_channels (
		username ` + st.keyedTextType + ` NOT NULL,
		channel ` + st.keyedTextType + ` NOT NULL,
		updated_at ` + st.dateTimeType + ` NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (username)
	);`); err != nil {
		return fmt.Errorf("create user_channels table: %w", err)
	}

	if _, err := conn.Exec(`
	CREATE TABLE IF NOT EXISTS read_receipts (
		username ` + st.keyedTextType + ` NOT NULL,
		message_id INTEGER NOT NULL,
		read_at ` + st.dateTimeType + ` NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (username, message_id)
	);`); err != nil {
		return fmt.Errorf("create read_receipts table: %w", err)
	}

	return nil
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	return columnExistsConn(migrationConn{db: db}, table, column)
}

func columnExistsConn(conn migrationConn, table, column string) (bool, error) {
	var exists int
	var err error
	switch getDBDialect(conn.db) {
	case DialectPostgres:
		err = conn.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?`, table, column).Scan(&exists)
	case DialectMySQL:
		err = conn.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`, table, column).Scan(&exists)
	default:
		err = conn.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM pragma_table_info(%q) WHERE name=?`, table), column).Scan(&exists)
	}
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func tableExists(db *sql.DB, table string) (bool, error) {
	var exists int
	var err error
	switch getDBDialect(db) {
	case DialectPostgres:
		err = dbQueryRow(db, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?`, table).Scan(&exists)
	case DialectMySQL:
		err = dbQueryRow(db, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, table).Scan(&exists)
	default:
		err = dbQueryRow(db, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&exists)
	}
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func verifySchema(db *sql.DB) error {
	requiredTables := []string{
		"messages",
		"user_message_state",
		"ban_history",
		"message_reactions",
		"user_channels",
		"read_receipts",
		"schema_version",
	}
	for _, name := range requiredTables {
		ok, err := tableExists(db, name)
		if err != nil {
			return fmt.Errorf("verify table %q: %w", name, err)
		}
		if !ok {
			return fmt.Errorf("required table %q is missing", name)
		}
	}

	hasExpires, err := columnExists(db, "ban_history", "expires_at")
	if err != nil {
		return fmt.Errorf("verify ban_history.expires_at: %w", err)
	}
	if !hasExpires {
		return fmt.Errorf("required column ban_history.expires_at is missing")
	}

	version, err := readSchemaVersion(db)
	if err != nil {
		return fmt.Errorf("verify schema version: %w", err)
	}
	if version < currentSchemaVersion {
		return fmt.Errorf("schema version %d is below required %d", version, currentSchemaVersion)
	}

	return nil
}

// CreateSchema applies schema migrations and terminates the process on failure.
// Tests and legacy callers use this wrapper; production startup should call MigrateSchema directly.
func CreateSchema(db *sql.DB) {
	if err := MigrateSchema(db); err != nil {
		log.Fatal(err)
	}
}
