package server

import (
	"crypto/hmac"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Cod-e-Codes/marchat/shared"

	"github.com/gorilla/websocket"
)

// CheckOrigin policy:
//   - Empty Origin is allowed because terminal/TUI clients do not send one.
//   - Same-host and localhost/loopback are allowed for dev and browser-based admin panels.
//   - All other origins are rejected. If you need to allow specific external
//     origins (e.g. a web frontend on a different domain), add them to the
//     allowlist below or set MARCHAT_ALLOWED_ORIGINS.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		host := r.Host
		if host == "" {
			host = r.Header.Get("Host")
		}
		if strings.Contains(origin, host) {
			return true
		}
		for _, local := range []string{"localhost", "127.0.0.1", "[::1]"} {
			if strings.Contains(origin, local) {
				return true
			}
		}
		log.Printf("WebSocket origin rejected: %s (host: %s)", origin, host)
		return false
	},
}

var (
	recentMessagesCache      []shared.Message
	recentMessagesCachedAt   time.Time
	recentMessagesCacheTTL   = 2 * time.Second
	recentMessagesCacheMutex sync.RWMutex
)

func invalidateRecentMessagesCache() {
	recentMessagesCacheMutex.Lock()
	defer recentMessagesCacheMutex.Unlock()
	recentMessagesCache = nil
	recentMessagesCachedAt = time.Time{}
}

func maxMessageRetention() int {
	maxMsgs := 1000
	if raw := strings.TrimSpace(os.Getenv("MARCHAT_MESSAGE_RETENTION_MAX")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			maxMsgs = n
		}
	}
	return maxMsgs
}

func messageTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("MARCHAT_MESSAGE_TTL_HOURS"))
	if raw == "" {
		return 0
	}
	h, err := strconv.Atoi(raw)
	if err != nil || h <= 0 {
		return 0
	}
	return time.Duration(h) * time.Hour
}

type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type UserList struct {
	Users []string `json:"users"`
}

// getClientIP extracts the real IP address from the request
func getClientIP(r *http.Request) string {
	// Check for forwarded headers first (for proxy/reverse proxy scenarios)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if comma := strings.Index(xff, ","); comma != -1 {
			return strings.TrimSpace(xff[:comma])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// Fall back to remote address
	if r.RemoteAddr != "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			return host
		}
		return r.RemoteAddr
	}
	return "unknown"
}

func CreateSchema(db *sql.DB) {
	dialect := getDBDialect(db)
	idColumn := "id INTEGER PRIMARY KEY AUTOINCREMENT"
	boolDefault := "BOOLEAN DEFAULT 0"
	dateTimeType := "DATETIME"
	blobType := "BLOB"
	textType := "TEXT"
	keyedTextType := "TEXT"
	switch dialect {
	case DialectPostgres:
		idColumn = "id BIGSERIAL PRIMARY KEY"
		boolDefault = "BOOLEAN DEFAULT FALSE"
		dateTimeType = "TIMESTAMPTZ"
		blobType = "BYTEA"
	case DialectMySQL:
		idColumn = "id BIGINT PRIMARY KEY AUTO_INCREMENT"
		boolDefault = "BOOLEAN DEFAULT FALSE"
		dateTimeType = "DATETIME"
		blobType = "LONGBLOB"
		textType = "LONGTEXT"
		keyedTextType = "VARCHAR(191)"
	}

	// First, create the basic messages table if it doesn't exist
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
		recipient %s
	);`, idColumn, textType, textType, dateTimeType, boolDefault, boolDefault, boolDefault, boolDefault, blobType, blobType, textType)

	_, err := dbExec(db, basicSchema)
	if err != nil {
		log.Fatal("failed to create basic schema:", err)
	}

	// Migrations: add columns if they don't exist
	migrations := []struct {
		column string
		ddl    string
	}{
		{"message_id", `ALTER TABLE messages ADD COLUMN message_id INTEGER DEFAULT 0`},
		{"edited", `ALTER TABLE messages ADD COLUMN edited BOOLEAN DEFAULT 0`},
		{"deleted", `ALTER TABLE messages ADD COLUMN deleted BOOLEAN DEFAULT 0`},
		{"pinned", `ALTER TABLE messages ADD COLUMN pinned BOOLEAN DEFAULT 0`},
	}

	for _, m := range migrations {
		var exists int
		switch dialect {
		case DialectPostgres:
			err = dbQueryRow(db, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'messages' AND column_name = ?`, m.column).Scan(&exists)
		case DialectMySQL:
			err = dbQueryRow(db, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'messages' AND column_name = ?`, m.column).Scan(&exists)
		default:
			err = dbQueryRow(db, `SELECT COUNT(*) FROM pragma_table_info('messages') WHERE name=?`, m.column).Scan(&exists)
		}
		if err != nil {
			log.Printf("Warning: failed to check for %s column: %v", m.column, err)
			continue
		}
		if exists == 0 {
			_, err = dbExec(db, m.ddl)
			if err != nil {
				log.Printf("Warning: failed to add %s column: %v", m.column, err)
			} else {
				log.Printf("Added %s column to messages table", m.column)
			}
		}
	}

	// Create user_message_state table
	userStateSchema := `
	CREATE TABLE IF NOT EXISTS user_message_state (
		username ` + keyedTextType + ` PRIMARY KEY,
		last_message_id INTEGER NOT NULL DEFAULT 0,
		last_seen ` + dateTimeType + ` NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = dbExec(db, userStateSchema)
	if err != nil {
		log.Fatal("failed to create user_message_state table:", err)
	}

	// Create ban_history table
	banHistoryID := "id INTEGER PRIMARY KEY AUTOINCREMENT"
	switch dialect {
	case DialectPostgres:
		banHistoryID = "id BIGSERIAL PRIMARY KEY"
	case DialectMySQL:
		banHistoryID = "id BIGINT PRIMARY KEY AUTO_INCREMENT"
	}
	banHistorySchema := `
	CREATE TABLE IF NOT EXISTS ban_history (
		` + banHistoryID + `,
		username ` + keyedTextType + ` NOT NULL,
		banned_at ` + dateTimeType + ` NOT NULL DEFAULT CURRENT_TIMESTAMP,
		unbanned_at ` + dateTimeType + `,
		banned_by ` + keyedTextType + ` NOT NULL
	);`

	_, err = dbExec(db, banHistorySchema)
	if err != nil {
		log.Printf("Warning: failed to create ban_history table: %v", err)
	}

	// Create indexes for performance (MySQL needs a prefix length when indexing LONGTEXT)
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
			// MySQL does not support "CREATE INDEX IF NOT EXISTS ..." (syntax error).
			q = strings.Replace(index, "IF NOT EXISTS ", "", 1)
		}
		_, err = dbExec(db, q)
		if err != nil {
			if dialect == DialectMySQL && isMySQLDuplicateKeyName(err) {
				continue
			}
			log.Printf("Warning: failed to create index: %v", err)
		}
	}

	// Migration: Update existing messages to have message_id = id
	_, err = dbExec(db, `UPDATE messages SET message_id = id WHERE message_id = 0 OR message_id IS NULL`)
	if err != nil {
		log.Printf("Warning: failed to migrate existing messages: %v", err)
	} else {
		log.Printf("Successfully migrated existing messages")
	}

	// Reactions table (durable reactions across reconnects)
	_, err = dbExec(db, `
	CREATE TABLE IF NOT EXISTS message_reactions (
		`+idColumn+`,
		message_id INTEGER NOT NULL,
		username `+keyedTextType+` NOT NULL,
		emoji `+keyedTextType+` NOT NULL,
		created_at `+dateTimeType+` NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(message_id, username, emoji)
	);`)
	if err != nil {
		log.Printf("Warning: failed to create message_reactions table: %v", err)
	}

	// Channel memberships table (durable memberships across reconnects)
	_, err = dbExec(db, `
	CREATE TABLE IF NOT EXISTS user_channels (
		username `+keyedTextType+` NOT NULL,
		channel `+keyedTextType+` NOT NULL,
		updated_at `+dateTimeType+` NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (username)
	);`)
	if err != nil {
		log.Printf("Warning: failed to create user_channels table: %v", err)
	}

	// Read receipt state tracking
	_, err = dbExec(db, `
	CREATE TABLE IF NOT EXISTS read_receipts (
		username `+keyedTextType+` NOT NULL,
		message_id INTEGER NOT NULL,
		read_at `+dateTimeType+` NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (username, message_id)
	);`)
	if err != nil {
		log.Printf("Warning: failed to create read_receipts table: %v", err)
	}
}

func InsertMessage(db *sql.DB, msg shared.Message) (int64, error) {
	var id int64
	if supportsLastInsertID(db) {
		result, err := dbExec(db, `INSERT INTO messages (sender, content, created_at, is_encrypted, recipient) VALUES (?, ?, ?, ?, ?)`,
			msg.Sender, msg.Content, msg.CreatedAt, msg.Encrypted, msg.Recipient)
		if err != nil {
			log.Println("Insert error:", err)
			return 0, fmt.Errorf("insert message: %w", err)
		}
		var errID error
		id, errID = result.LastInsertId()
		if errID != nil {
			log.Println("Error getting last insert ID:", errID)
			return 0, fmt.Errorf("last insert id: %w", errID)
		}
	} else {
		err := dbQueryRow(db, `INSERT INTO messages (sender, content, created_at, is_encrypted, recipient) VALUES (?, ?, ?, ?, ?) RETURNING id`,
			msg.Sender, msg.Content, msg.CreatedAt, msg.Encrypted, msg.Recipient).Scan(&id)
		if err != nil {
			log.Println("Insert returning error:", err)
			return 0, fmt.Errorf("insert message returning: %w", err)
		}
	}

	_, err := dbExec(db, `UPDATE messages SET message_id = ? WHERE id = ?`, id, id)
	if err != nil {
		log.Println("Error updating message_id:", err)
	}

	enforceMessageRetention(db)
	invalidateRecentMessagesCache()
	return id, nil
}

// InsertEncryptedMessage stores an encrypted message in the database
func InsertEncryptedMessage(db *sql.DB, encryptedMsg *shared.EncryptedMessage) error {
	var id int64
	if supportsLastInsertID(db) {
		result, err := dbExec(db, `INSERT INTO messages (sender, content, created_at, is_encrypted, encrypted_data, nonce, recipient) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			encryptedMsg.Sender, encryptedMsg.Content, encryptedMsg.CreatedAt,
			encryptedMsg.IsEncrypted, encryptedMsg.Encrypted, encryptedMsg.Nonce, encryptedMsg.Recipient)
		if err != nil {
			log.Println("Insert encrypted message error:", err)
			return fmt.Errorf("insert encrypted message: %w", err)
		}
		var errID error
		id, errID = result.LastInsertId()
		if errID != nil {
			log.Println("Error getting last insert ID for encrypted message:", errID)
		}
	} else {
		err := dbQueryRow(db, `INSERT INTO messages (sender, content, created_at, is_encrypted, encrypted_data, nonce, recipient) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id`,
			encryptedMsg.Sender, encryptedMsg.Content, encryptedMsg.CreatedAt,
			encryptedMsg.IsEncrypted, encryptedMsg.Encrypted, encryptedMsg.Nonce, encryptedMsg.Recipient).Scan(&id)
		if err != nil {
			log.Println("Insert encrypted message returning error:", err)
			return fmt.Errorf("insert encrypted message: %w", err)
		}
	}

	if id > 0 {
		_, err := dbExec(db, `UPDATE messages SET message_id = ? WHERE id = ?`, id, id)
		if err != nil {
			log.Println("Error updating message_id for encrypted message:", err)
		}
	}

	enforceMessageRetention(db)
	invalidateRecentMessagesCache()
	return nil
}

func GetRecentMessages(db *sql.DB) []shared.Message {
	recentMessagesCacheMutex.RLock()
	if time.Since(recentMessagesCachedAt) <= recentMessagesCacheTTL && len(recentMessagesCache) > 0 {
		out := make([]shared.Message, len(recentMessagesCache))
		copy(out, recentMessagesCache)
		recentMessagesCacheMutex.RUnlock()
		return out
	}
	recentMessagesCacheMutex.RUnlock()

	rows, err := dbQuery(db, `SELECT sender, content, created_at, is_encrypted, message_id, COALESCE(edited, 0), COALESCE(deleted, 0), COALESCE(recipient, '') FROM messages ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		log.Println("Query error:", err)
		return nil
	}
	defer rows.Close()

	var messages []shared.Message
	for rows.Next() {
		var msg shared.Message
		var isEncrypted, edited, deleted bool
		err := rows.Scan(&msg.Sender, &msg.Content, &msg.CreatedAt, &isEncrypted, &msg.MessageID, &edited, &deleted, &msg.Recipient)
		if err == nil {
			msg.Encrypted = isEncrypted
			msg.Edited = edited
			if strings.TrimSpace(msg.Recipient) != "" {
				msg.Type = shared.DirectMessage
			}
			if deleted {
				msg.Type = shared.DeleteMessage
			}
			messages = append(messages, msg)
		}
	}

	// CRITICAL FIX: Always sort messages by timestamp for consistent chronological display
	// Note: SQL query fetches newest messages first (DESC), but we sort chronologically (ASC) for display
	sortMessagesByTimestamp(messages)

	recentMessagesCacheMutex.Lock()
	recentMessagesCache = make([]shared.Message, len(messages))
	copy(recentMessagesCache, messages)
	recentMessagesCachedAt = time.Now()
	recentMessagesCacheMutex.Unlock()

	return messages
}

func enforceMessageRetention(db *sql.DB) {
	maxMessages := maxMessageRetention()
	ttl := messageTTL()

	if ttl > 0 {
		for {
			// Nested subquery keeps MySQL happy (LIMIT inside IN/NOT IN is invalid otherwise).
			result, err := dbExec(db, `DELETE FROM messages WHERE id IN (
				SELECT id FROM (
					SELECT id FROM messages WHERE created_at < ? ORDER BY id ASC LIMIT 500
				) AS ttl_batch
			)`, time.Now().Add(-ttl))
			if err != nil {
				log.Printf("Error enforcing TTL retention: %v", err)
				break
			}
			rows, _ := result.RowsAffected()
			if rows == 0 {
				break
			}
		}
	}

	_, err := dbExec(db, `DELETE FROM messages WHERE id NOT IN (
		SELECT id FROM (
			SELECT id FROM messages ORDER BY id DESC LIMIT ?
		) AS keep_batch
	)`, maxMessages)
	if err != nil {
		log.Printf("Error enforcing message cap: %v", err)
	}
}

// GetRecentMessagesForUser returns personalized message history for a specific user
func GetRecentMessagesForUser(db *sql.DB, username string, defaultLimit int, banGapsHistory bool) ([]shared.Message, int64) {
	lowerUsername := strings.ToLower(username)

	// Get user's last seen message ID
	lastMessageID, err := getUserLastMessageID(db, lowerUsername)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error getting last message ID for user %s: %v", username, err)
		// Fall back to recent messages for new users or on error
		messages := GetRecentMessages(db)
		sortMessagesByTimestamp(messages) // Ensure consistent ordering
		return messages, 0
	}

	var messages []shared.Message

	if lastMessageID == 0 {
		// New user or no history - get recent messages
		messages = GetRecentMessages(db)
	} else {
		// Returning user - get messages after their last seen ID
		messages = GetMessagesAfter(db, lastMessageID, defaultLimit)

		// If they have few new messages, combine with recent history
		if len(messages) < defaultLimit/2 {
			recentMessages := GetRecentMessages(db)
			// Combine recent messages with new messages, avoiding duplicates
			existingIDs := make(map[string]bool)
			for _, msg := range messages {
				key := msg.Sender + ":" + msg.Content + ":" + msg.CreatedAt.Format("2006-01-02 15:04:05")
				existingIDs[key] = true
			}

			for _, msg := range recentMessages {
				key := msg.Sender + ":" + msg.Content + ":" + msg.CreatedAt.Format("2006-01-02 15:04:05")
				if !existingIDs[key] && len(messages) < defaultLimit {
					messages = append(messages, msg)
				}
			}
		}
	}

	// CRITICAL FIX: Always sort messages by timestamp for consistent chronological display
	// Note: SQL queries fetch newest messages first (DESC), but we sort chronologically (ASC) for display
	sortMessagesByTimestamp(messages)

	// Filter to messages visible to this user.
	// Public/channel messages have empty recipient.
	// DMs are visible only to sender and recipient.
	visible := make([]shared.Message, 0, len(messages))
	for _, msg := range messages {
		recipient := strings.ToLower(strings.TrimSpace(msg.Recipient))
		if recipient == "" {
			visible = append(visible, msg)
			continue
		}
		sender := strings.ToLower(strings.TrimSpace(msg.Sender))
		if sender == lowerUsername || recipient == lowerUsername {
			visible = append(visible, msg)
		}
	}
	messages = visible

	// Filter messages during ban periods if feature is enabled
	if banGapsHistory {
		banPeriods, err := getUserBanPeriods(db, lowerUsername)
		if err != nil {
			log.Printf("Warning: failed to get ban periods for user %s: %v", username, err)
		} else if len(banPeriods) > 0 {
			// Filter out messages sent during ban periods
			filteredMessages := make([]shared.Message, 0, len(messages))
			for _, msg := range messages {
				if !isMessageInBanPeriod(msg.CreatedAt, banPeriods) {
					filteredMessages = append(filteredMessages, msg)
				}
			}
			messages = filteredMessages
			log.Printf("Filtered %d messages for user %s due to ban history gaps", len(messages), username)
		}
	}

	// Update user's last seen message ID
	if len(messages) > 0 {
		latestID := getLatestMessageID(db)
		if latestID > 0 {
			err = setUserLastMessageID(db, lowerUsername, latestID)
			if err != nil {
				log.Printf("Warning: failed to update last message ID for user %s: %v", username, err)
			}
		}
	}

	return messages, lastMessageID
}

// GetMessagesAfter retrieves messages with ID > lastMessageID
func GetMessagesAfter(db *sql.DB, lastMessageID int64, limit int) []shared.Message {
	rows, err := dbQuery(db, `SELECT sender, content, created_at, is_encrypted, message_id, COALESCE(edited, 0), COALESCE(deleted, 0), COALESCE(recipient, '') FROM messages WHERE message_id > ? ORDER BY created_at DESC LIMIT ?`, lastMessageID, limit)
	if err != nil {
		log.Println("Query error in GetMessagesAfter:", err)
		return nil
	}
	defer rows.Close()

	var messages []shared.Message
	for rows.Next() {
		var msg shared.Message
		var isEncrypted, edited, deleted bool
		err := rows.Scan(&msg.Sender, &msg.Content, &msg.CreatedAt, &isEncrypted, &msg.MessageID, &edited, &deleted, &msg.Recipient)
		if err == nil {
			msg.Encrypted = isEncrypted
			msg.Edited = edited
			if strings.TrimSpace(msg.Recipient) != "" {
				msg.Type = shared.DirectMessage
			}
			if deleted {
				msg.Type = shared.DeleteMessage
			}
			messages = append(messages, msg)
		}
	}

	// CRITICAL FIX: Always sort messages by timestamp for consistent chronological display
	// Note: SQL query fetches newest messages first (DESC), but we sort chronologically (ASC) for display
	sortMessagesByTimestamp(messages)
	return messages
}

// getUserLastMessageID queries user_message_state table
func getUserLastMessageID(db *sql.DB, username string) (int64, error) {
	var lastMessageID int64
	err := dbQueryRow(db, `SELECT last_message_id FROM user_message_state WHERE username = ?`, username).Scan(&lastMessageID)
	return lastMessageID, err
}

// setUserLastMessageID upserts user_message_state in a backend-compatible way
func setUserLastMessageID(db *sql.DB, username string, messageID int64) error {
	_, err := dbExec(db, upsertUserMessageStateSQL(db), username, messageID)
	return err
}

// getLatestMessageID returns MAX(id) from messages table
func getLatestMessageID(db *sql.DB) int64 {
	var latestID int64
	err := dbQueryRow(db, `SELECT MAX(id) FROM messages`).Scan(&latestID)
	if err != nil {
		// Handle empty table case
		return 0
	}
	return latestID
}

// clearUserMessageState deletes user's record from user_message_state
func clearUserMessageState(db *sql.DB, username string) error {
	_, err := dbExec(db, `DELETE FROM user_message_state WHERE username = ?`, username)
	return err
}

// recordBanEvent records a ban event in the ban_history table
func recordBanEvent(db *sql.DB, username, bannedBy string) error {
	_, err := dbExec(db, `INSERT INTO ban_history (username, banned_by) VALUES (?, ?)`, username, bannedBy)
	if err != nil {
		log.Printf("Warning: failed to record ban event for user %s: %v", username, err)
	}
	return err
}

// recordUnbanEvent records an unban event in the ban_history table
func recordUnbanEvent(db *sql.DB, username string) error {
	_, err := dbExec(db, `UPDATE ban_history SET unbanned_at = CURRENT_TIMESTAMP WHERE username = ? AND unbanned_at IS NULL`, username)
	if err != nil {
		log.Printf("Warning: failed to record unban event for user %s: %v", username, err)
	}
	return err
}

// getUserBanPeriods retrieves all ban periods for a user
func getUserBanPeriods(db *sql.DB, username string) ([]struct {
	BannedAt   time.Time
	UnbannedAt *time.Time
}, error) {
	rows, err := dbQuery(db, `SELECT banned_at, unbanned_at FROM ban_history WHERE username = ? ORDER BY banned_at ASC`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var periods []struct {
		BannedAt   time.Time
		UnbannedAt *time.Time
	}

	for rows.Next() {
		var bannedAt time.Time
		var unbannedAt *time.Time
		err := rows.Scan(&bannedAt, &unbannedAt)
		if err != nil {
			log.Printf("Warning: failed to scan ban period for user %s: %v", username, err)
			continue
		}
		periods = append(periods, struct {
			BannedAt   time.Time
			UnbannedAt *time.Time
		}{bannedAt, unbannedAt})
	}

	return periods, nil
}

// isMessageInBanPeriod checks if a message was sent during a user's ban period
func isMessageInBanPeriod(messageTime time.Time, banPeriods []struct {
	BannedAt   time.Time
	UnbannedAt *time.Time
}) bool {
	for _, period := range banPeriods {
		// If unbanned_at is nil, the user is still banned
		if period.UnbannedAt == nil {
			if messageTime.After(period.BannedAt) {
				return true
			}
		} else {
			// Check if message was sent during the ban period
			if messageTime.After(period.BannedAt) && messageTime.Before(*period.UnbannedAt) {
				return true
			}
		}
	}
	return false
}

// sortMessagesByTimestamp ensures messages are displayed in chronological order
// This provides server-side protection against ordering issues
func sortMessagesByTimestamp(messages []shared.Message) {
	sort.Slice(messages, func(i, j int) bool {
		// Primary sort: by timestamp
		if !messages[i].CreatedAt.Equal(messages[j].CreatedAt) {
			return messages[i].CreatedAt.Before(messages[j].CreatedAt)
		}
		// Secondary sort: by sender for deterministic ordering when timestamps are identical
		if messages[i].Sender != messages[j].Sender {
			return messages[i].Sender < messages[j].Sender
		}
		// Tertiary sort: by content for full deterministic ordering
		return messages[i].Content < messages[j].Content
	})
}

func ClearMessages(db *sql.DB) error {
	_, err := dbExec(db, `DELETE FROM messages`)
	invalidateRecentMessagesCache()
	return err
}

func EditMessage(db *sql.DB, messageID int64, sender, newContent string, encrypted bool) error {
	result, err := dbExec(db,
		`UPDATE messages SET content = ?, edited = 1, is_encrypted = ? WHERE message_id = ? AND sender = ?`,
		newContent, encrypted, messageID, sender)
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message not found or you are not the sender")
	}
	invalidateRecentMessagesCache()
	return nil
}

func DeleteMessage(db *sql.DB, messageID int64, sender string, isAdmin bool) error {
	var query string
	var args []interface{}
	if isAdmin {
		query = `UPDATE messages SET content = '[deleted]', deleted = 1, is_encrypted = 0 WHERE message_id = ?`
		args = []interface{}{messageID}
	} else {
		query = `UPDATE messages SET content = '[deleted]', deleted = 1, is_encrypted = 0 WHERE message_id = ? AND sender = ?`
		args = []interface{}{messageID, sender}
	}
	result, err := dbExec(db, query, args...)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message not found or you are not the sender")
	}
	invalidateRecentMessagesCache()
	return nil
}

func SearchMessages(db *sql.DB, query string, limit int) []shared.Message {
	if limit <= 0 {
		limit = 20
	}
	rows, err := dbQuery(db,
		`SELECT sender, content, created_at, is_encrypted, message_id FROM messages WHERE content LIKE ? AND deleted = 0 ORDER BY created_at DESC LIMIT ?`,
		"%"+query+"%", limit)
	if err != nil {
		log.Println("Search query error:", err)
		return nil
	}
	defer rows.Close()
	var messages []shared.Message
	for rows.Next() {
		var msg shared.Message
		var isEncrypted bool
		err := rows.Scan(&msg.Sender, &msg.Content, &msg.CreatedAt, &isEncrypted, &msg.MessageID)
		if err == nil {
			msg.Encrypted = isEncrypted
			messages = append(messages, msg)
		}
	}
	return messages
}

func TogglePinMessage(db *sql.DB, messageID int64) (bool, error) {
	var currentlyPinned bool
	err := dbQueryRow(db, `SELECT COALESCE(pinned, 0) FROM messages WHERE message_id = ?`, messageID).Scan(&currentlyPinned)
	if err != nil {
		return false, fmt.Errorf("message not found: %w", err)
	}
	newPinned := !currentlyPinned
	_, err = dbExec(db, `UPDATE messages SET pinned = ? WHERE message_id = ?`, newPinned, messageID)
	if err != nil {
		return false, fmt.Errorf("toggle pin: %w", err)
	}
	return newPinned, nil
}

func GetPinnedMessages(db *sql.DB) []shared.Message {
	rows, err := dbQuery(db, `SELECT sender, content, created_at, message_id FROM messages WHERE pinned = 1 ORDER BY created_at DESC`)
	if err != nil {
		log.Println("Pinned messages query error:", err)
		return nil
	}
	defer rows.Close()
	var messages []shared.Message
	for rows.Next() {
		var msg shared.Message
		err := rows.Scan(&msg.Sender, &msg.Content, &msg.CreatedAt, &msg.MessageID)
		if err == nil {
			messages = append(messages, msg)
		}
	}
	return messages
}

// BackupDatabase creates a backup of the current database
func BackupDatabase(dbPath string) (string, error) {
	// Generate backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupFilename := fmt.Sprintf("marchat_backup_%s.db", timestamp)

	// Get directory of the original database
	dbDir := filepath.Dir(dbPath)
	backupPath := filepath.Join(dbDir, backupFilename)

	// Open the database connection for backup
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", fmt.Errorf("failed to open database for backup: %v", err)
	}
	defer db.Close()

	// For WAL mode, we need to checkpoint the WAL file to ensure all data is in the main file
	// This is safe to do while the database is in use
	_, err = dbExec(db, "PRAGMA wal_checkpoint(FULL);")
	if err != nil {
		log.Printf("Warning: WAL checkpoint failed during backup: %v", err)
		// Continue with backup even if checkpoint fails
	}

	// Use SQLite's built-in backup functionality
	// This ensures we get a consistent snapshot even with WAL mode
	backupDB, err := sql.Open("sqlite", backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup database: %v", err)
	}
	defer backupDB.Close()

	// Execute VACUUM INTO to create a clean backup
	// This creates a complete, consistent copy of the database
	stmt := "VACUUM INTO " + sqliteQuoteLiteral(backupPath) + ";"
	_, err = dbExec(db, stmt)
	if err != nil {
		return "", fmt.Errorf("failed to create database backup: %v", err)
	}

	return backupFilename, nil
}

// sqliteQuoteLiteral returns s as a single-quoted SQLite string literal (embedded ' doubled).
func sqliteQuoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// GetDatabaseStats returns statistics about the database
func GetDatabaseStats(db *sql.DB) (string, error) {
	var stats strings.Builder

	// Count messages
	var messageCount int
	err := dbQueryRow(db, "SELECT COUNT(*) FROM messages").Scan(&messageCount)
	if err != nil {
		return "", fmt.Errorf("failed to count messages: %v", err)
	}

	// Count unique users
	var userCount int
	err = dbQueryRow(db, "SELECT COUNT(DISTINCT sender) FROM messages WHERE sender != 'System'").Scan(&userCount)
	if err != nil {
		return "", fmt.Errorf("failed to count users: %v", err)
	}

	// Get oldest and newest message dates
	var oldestDate, newestDate sql.NullString
	err = dbQueryRow(db, "SELECT MIN(created_at), MAX(created_at) FROM messages").Scan(&oldestDate, &newestDate)
	if err != nil {
		return "", fmt.Errorf("failed to get date range: %v", err)
	}

	stats.WriteString("Database Statistics:\n")
	stats.WriteString(fmt.Sprintf("  Total Messages: %d\n", messageCount))
	stats.WriteString(fmt.Sprintf("  Unique Users: %d\n", userCount))
	if oldestDate.Valid && newestDate.Valid {
		stats.WriteString(fmt.Sprintf("  Date Range: %s to %s\n", oldestDate.String, newestDate.String))
	}

	return stats.String(), nil
}

func (h *Hub) broadcastUserList() {
	h.clientsMutex.RLock()
	usernames := []string{}
	for client := range h.clients {
		if client.username != "" {
			usernames = append(usernames, client.username)
		}
	}
	sort.Strings(usernames)
	userList := UserList{Users: usernames}
	payload, _ := json.Marshal(userList)
	msg := WSMessage{Type: "userlist", Data: payload}
	for client := range h.clients {
		select {
		case client.send <- msg:
		default:
			log.Printf("Dropping user list update for client %s (send buffer full)", client.username)
		}
	}
	h.clientsMutex.RUnlock()
}

// formatWSClose returns RFC 6455 close frame application data: 2-byte status (big-endian)
// plus optional UTF-8 reason, total length at most 123 bytes.
func formatWSClose(code int, text string) []byte {
	const maxPayload = 123
	maxText := maxPayload - 2
	if maxText < 0 {
		maxText = 0
	}
	text = truncateUTF8CloseReason(text, maxText)
	return websocket.FormatCloseMessage(code, text)
}

func truncateUTF8CloseReason(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	s = s[:maxBytes]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

type adminAuth struct {
	admins   map[string]struct{}
	adminKey string
}

func ServeWs(hub *Hub, db *sql.DB, adminList []string, adminKey string, banGapsHistory bool, maxFileBytes int64, dbPath string) http.HandlerFunc {
	auth := adminAuth{admins: make(map[string]struct{}), adminKey: adminKey}
	for _, u := range adminList {
		auth.admins[strings.ToLower(u)] = struct{}{}
	}

	// Parse allowed users from environment variable (username allowlist)
	var allowedUsers map[string]struct{}
	if allowedUsersEnv := os.Getenv("MARCHAT_ALLOWED_USERS"); allowedUsersEnv != "" {
		allowedUsers = make(map[string]struct{})
		for _, u := range strings.Split(allowedUsersEnv, ",") {
			username := strings.TrimSpace(u)
			if username != "" {
				allowedUsers[strings.ToLower(username)] = struct{}{}
			}
		}
		log.Printf("Username allowlist enabled with %d allowed users", len(allowedUsers))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:", err)
			return
		}
		// Expect handshake as first message
		var hs shared.Handshake
		err = conn.ReadJSON(&hs)
		if err != nil {
			if err := conn.WriteMessage(websocket.CloseMessage, formatWSClose(websocket.CloseProtocolError, "Invalid handshake")); err != nil {
				log.Printf("WriteMessage error: %v", err)
			}
			conn.Close()
			return
		}
		username := strings.TrimSpace(hs.Username)
		if username == "" {
			if err := conn.WriteMessage(websocket.CloseMessage, formatWSClose(websocket.ClosePolicyViolation, "Username required")); err != nil {
				log.Printf("WriteMessage error: %v", err)
			}
			conn.Close()
			return
		}

		// Validate username format
		if err := validateUsername(username); err != nil {
			SecurityLogger.Warn("Invalid username attempt", map[string]interface{}{
				"username": username,
				"error":    err.Error(),
				"ip":       getClientIP(r),
			})
			if err := conn.WriteMessage(websocket.CloseMessage, formatWSClose(websocket.ClosePolicyViolation, "Invalid username: "+err.Error())); err != nil {
				log.Printf("WriteMessage error: %v", err)
			}
			conn.Close()
			return
		}

		lu := strings.ToLower(username)

		// Check username allowlist if enabled
		if allowedUsers != nil {
			if _, allowed := allowedUsers[lu]; !allowed {
				SecurityLogger.Warn("Username not in allowlist", map[string]interface{}{
					"username": username,
					"ip":       getClientIP(r),
				})
				log.Printf("User '%s' (IP: %s) rejected - not in allowed users list", username, getClientIP(r))
				if err := conn.WriteMessage(websocket.CloseMessage, formatWSClose(websocket.ClosePolicyViolation, "Username not allowed on this server")); err != nil {
					log.Printf("WriteMessage error: %v", err)
				}
				conn.Close()
				return
			}
		}
		isAdmin := false
		if hs.Admin {
			if _, ok := auth.admins[lu]; !ok {
				if err := conn.WriteMessage(websocket.CloseMessage, formatWSClose(websocket.ClosePolicyViolation, "Not an admin user")); err != nil {
					log.Printf("WriteMessage error: %v", err)
				}
				conn.Close()
				return
			}
			if !hmac.Equal([]byte(hs.AdminKey), []byte(auth.adminKey)) {
				failMsg, _ := json.Marshal(map[string]string{"reason": "invalid admin key"})
				if err := conn.WriteJSON(WSMessage{Type: "auth_failed", Data: failMsg}); err != nil {
					log.Printf("WriteMessage error: %v", err)
				}
				if err := conn.WriteMessage(websocket.CloseMessage, formatWSClose(websocket.ClosePolicyViolation, "Invalid admin key")); err != nil {
					log.Printf("WriteMessage error: %v", err)
				}
				conn.Close()
				return
			}
			isAdmin = true
		}

		// Extract IP address
		ipAddr := getClientIP(r)

		// Atomically reserve username before creating client/session state.
		if !hub.TryReserveUsername(username) {
			log.Printf("Duplicate username attempt blocked: '%s' (IP: %s)", username, ipAddr)
			if err := conn.WriteMessage(websocket.CloseMessage, formatWSClose(websocket.ClosePolicyViolation, "Username already taken - please choose a different username")); err != nil {
				log.Printf("WriteMessage error: %v", err)
			}
			conn.Close()
			return
		}
		usernameReserved := true
		defer func() {
			if usernameReserved {
				hub.ReleaseUsername(username)
			}
		}()

		// Check if user is banned
		if hub.IsUserBanned(username) {
			log.Printf("Banned user '%s' (IP: %s) attempted to connect", username, ipAddr)
			if err := conn.WriteMessage(websocket.CloseMessage, formatWSClose(websocket.ClosePolicyViolation, "You are banned from this server")); err != nil {
				log.Printf("WriteMessage error: %v", err)
			}
			conn.Close()
			return
		}

		client := &Client{
			hub:                  hub,
			conn:                 conn,
			send:                 make(chan interface{}, 256),
			db:                   db,
			username:             username,
			isAdmin:              isAdmin,
			ipAddr:               ipAddr,
			pluginCommandHandler: hub.pluginCommandHandler,
			maxFileBytes:         maxFileBytes,
			dbPath:               dbPath,
		}
		log.Printf("Client %s connected (admin=%v, IP: %s)", username, isAdmin, ipAddr)
		hub.register <- client
		usernameReserved = false

		if persistedChannel := LoadUserChannel(db, username); persistedChannel != "" && persistedChannel != "general" {
			hub.leaveChannel(client, "general")
			hub.joinChannel(client, persistedChannel)
		}

		// Start writePump before enqueuing history so the channel has a
		// consumer.  Without this, replaying messages + reactions + receipts
		// into the 256-capacity channel can block the handler goroutine
		// indefinitely when total items exceed the buffer.
		go client.writePump()

		msgs, _ := GetRecentMessagesForUser(db, username, 50, banGapsHistory)
		messageIDs := make([]int64, 0, len(msgs))
		for _, msg := range msgs {
			select {
			case client.send <- msg:
			default:
				log.Printf("Dropping history message for new client %s (send buffer full)", username)
			}
			if msg.MessageID > 0 {
				messageIDs = append(messageIDs, msg.MessageID)
			}
		}
		for _, reactionMsg := range LoadReactionsForMessages(db, messageIDs) {
			select {
			case client.send <- reactionMsg:
			default:
				log.Printf("Dropping reaction replay for new client %s (send buffer full)", username)
			}
		}
		for _, receiptMsg := range LoadReadReceiptsForMessages(db, username, messageIDs) {
			select {
			case client.send <- receiptMsg:
			default:
				log.Printf("Dropping read receipt replay for new client %s (send buffer full)", username)
			}
		}
		hub.broadcastUserList()

		go client.readPump()
	}
}
