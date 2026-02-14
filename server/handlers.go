package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow non-browser clients (terminal clients don't send Origin)
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		// Allow same-host connections (browser-based clients)
		host := r.Host
		if host == "" {
			host = r.Header.Get("Host")
		}
		// Accept if origin matches the server host
		if strings.Contains(origin, host) {
			return true
		}
		// Allow localhost/loopback origins for local development
		for _, local := range []string{"localhost", "127.0.0.1", "[::1]"} {
			if strings.Contains(origin, local) {
				return true
			}
		}
		log.Printf("WebSocket origin rejected: %s (host: %s)", origin, host)
		return false
	},
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
	// First, create the basic messages table if it doesn't exist
	basicSchema := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sender TEXT,
		content TEXT,
		created_at DATETIME,
		is_encrypted BOOLEAN DEFAULT 0,
		encrypted_data BLOB,
		nonce BLOB,
		recipient TEXT
	);`

	_, err := db.Exec(basicSchema)
	if err != nil {
		log.Fatal("failed to create basic schema:", err)
	}

	// Check if message_id column exists, if not add it
	var columnExists int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('messages') WHERE name='message_id'`).Scan(&columnExists)
	if err != nil {
		log.Printf("Warning: failed to check for message_id column: %v", err)
	}

	if columnExists == 0 {
		// Add message_id column to existing table
		_, err = db.Exec(`ALTER TABLE messages ADD COLUMN message_id INTEGER DEFAULT 0`)
		if err != nil {
			log.Printf("Warning: failed to add message_id column: %v", err)
		} else {
			log.Printf("Added message_id column to messages table")
		}
	}

	// Create user_message_state table
	userStateSchema := `
	CREATE TABLE IF NOT EXISTS user_message_state (
		username TEXT PRIMARY KEY,
		last_message_id INTEGER NOT NULL DEFAULT 0,
		last_seen DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(userStateSchema)
	if err != nil {
		log.Fatal("failed to create user_message_state table:", err)
	}

	// Create ban_history table
	banHistorySchema := `
	CREATE TABLE IF NOT EXISTS ban_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL,
		banned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		unbanned_at DATETIME,
		banned_by TEXT NOT NULL
	);`

	_, err = db.Exec(banHistorySchema)
	if err != nil {
		log.Printf("Warning: failed to create ban_history table: %v", err)
	}

	// Create indexes for performance
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_messages_message_id ON messages(message_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_user_message_state_username ON user_message_state(username)`,
		`CREATE INDEX IF NOT EXISTS idx_ban_history_username ON ban_history(username)`,
		`CREATE INDEX IF NOT EXISTS idx_ban_history_banned_at ON ban_history(banned_at)`,
		`CREATE INDEX IF NOT EXISTS idx_ban_history_unbanned_at ON ban_history(unbanned_at)`,
	}

	for _, index := range indexes {
		_, err = db.Exec(index)
		if err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}

	// Migration: Update existing messages to have message_id = id
	_, err = db.Exec(`UPDATE messages SET message_id = id WHERE message_id = 0 OR message_id IS NULL`)
	if err != nil {
		log.Printf("Warning: failed to migrate existing messages: %v", err)
	} else {
		log.Printf("Successfully migrated existing messages")
	}
}

func InsertMessage(db *sql.DB, msg shared.Message) error {
	result, err := db.Exec(`INSERT INTO messages (sender, content, created_at, is_encrypted) VALUES (?, ?, ?, ?)`,
		msg.Sender, msg.Content, msg.CreatedAt, msg.Encrypted)
	if err != nil {
		log.Println("Insert error:", err)
		return fmt.Errorf("insert message: %w", err)
	}

	// Get the inserted ID and update message_id
	id, err := result.LastInsertId()
	if err != nil {
		log.Println("Error getting last insert ID:", err)
	} else {
		_, err = db.Exec(`UPDATE messages SET message_id = ? WHERE id = ?`, id, id)
		if err != nil {
			log.Println("Error updating message_id:", err)
		}
	}

	// Enforce message cap: keep only the most recent 1000 messages
	_, err = db.Exec(`DELETE FROM messages WHERE id NOT IN (SELECT id FROM messages ORDER BY id DESC LIMIT 1000)`)
	if err != nil {
		log.Println("Error enforcing message cap:", err)
	}
	return nil
}

// InsertEncryptedMessage stores an encrypted message in the database
func InsertEncryptedMessage(db *sql.DB, encryptedMsg *shared.EncryptedMessage) error {
	result, err := db.Exec(`INSERT INTO messages (sender, content, created_at, is_encrypted, encrypted_data, nonce, recipient) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		encryptedMsg.Sender, encryptedMsg.Content, encryptedMsg.CreatedAt,
		encryptedMsg.IsEncrypted, encryptedMsg.Encrypted, encryptedMsg.Nonce, encryptedMsg.Recipient)
	if err != nil {
		log.Println("Insert encrypted message error:", err)
		return fmt.Errorf("insert encrypted message: %w", err)
	}

	// Get the inserted ID and update message_id
	id, err := result.LastInsertId()
	if err != nil {
		log.Println("Error getting last insert ID for encrypted message:", err)
	} else {
		_, err = db.Exec(`UPDATE messages SET message_id = ? WHERE id = ?`, id, id)
		if err != nil {
			log.Println("Error updating message_id for encrypted message:", err)
		}
	}

	// Enforce message cap: keep only the most recent 1000 messages
	_, err = db.Exec(`DELETE FROM messages WHERE id NOT IN (SELECT id FROM messages ORDER BY id DESC LIMIT 1000)`)
	if err != nil {
		log.Println("Error enforcing message cap:", err)
	}
	return nil
}

func GetRecentMessages(db *sql.DB) []shared.Message {
	// FIXED: Changed ORDER BY created_at ASC to DESC to fetch newest messages first
	rows, err := db.Query(`SELECT sender, content, created_at, is_encrypted FROM messages ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		log.Println("Query error:", err)
		return nil
	}
	defer rows.Close()

	var messages []shared.Message
	for rows.Next() {
		var msg shared.Message
		var isEncrypted bool
		err := rows.Scan(&msg.Sender, &msg.Content, &msg.CreatedAt, &isEncrypted)
		if err == nil {
			msg.Encrypted = isEncrypted
			messages = append(messages, msg)
		}
	}

	// CRITICAL FIX: Always sort messages by timestamp for consistent chronological display
	// Note: SQL query fetches newest messages first (DESC), but we sort chronologically (ASC) for display
	sortMessagesByTimestamp(messages)
	return messages
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
	// FIXED: Changed ORDER BY created_at ASC to DESC to fetch newest messages first
	rows, err := db.Query(`SELECT sender, content, created_at, is_encrypted FROM messages WHERE message_id > ? ORDER BY created_at DESC LIMIT ?`, lastMessageID, limit)
	if err != nil {
		log.Println("Query error in GetMessagesAfter:", err)
		return nil
	}
	defer rows.Close()

	var messages []shared.Message
	for rows.Next() {
		var msg shared.Message
		var isEncrypted bool
		err := rows.Scan(&msg.Sender, &msg.Content, &msg.CreatedAt, &isEncrypted)
		if err == nil {
			msg.Encrypted = isEncrypted
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
	err := db.QueryRow(`SELECT last_message_id FROM user_message_state WHERE username = ?`, username).Scan(&lastMessageID)
	return lastMessageID, err
}

// setUserLastMessageID INSERT OR REPLACE into user_message_state
func setUserLastMessageID(db *sql.DB, username string, messageID int64) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO user_message_state (username, last_message_id, last_seen) VALUES (?, ?, CURRENT_TIMESTAMP)`, username, messageID)
	return err
}

// getLatestMessageID returns MAX(id) from messages table
func getLatestMessageID(db *sql.DB) int64 {
	var latestID int64
	err := db.QueryRow(`SELECT MAX(id) FROM messages`).Scan(&latestID)
	if err != nil {
		// Handle empty table case
		return 0
	}
	return latestID
}

// clearUserMessageState deletes user's record from user_message_state
func clearUserMessageState(db *sql.DB, username string) error {
	_, err := db.Exec(`DELETE FROM user_message_state WHERE username = ?`, username)
	return err
}

// recordBanEvent records a ban event in the ban_history table
func recordBanEvent(db *sql.DB, username, bannedBy string) error {
	_, err := db.Exec(`INSERT INTO ban_history (username, banned_by) VALUES (?, ?)`, username, bannedBy)
	if err != nil {
		log.Printf("Warning: failed to record ban event for user %s: %v", username, err)
	}
	return err
}

// recordUnbanEvent records an unban event in the ban_history table
func recordUnbanEvent(db *sql.DB, username string) error {
	_, err := db.Exec(`UPDATE ban_history SET unbanned_at = CURRENT_TIMESTAMP WHERE username = ? AND unbanned_at IS NULL`, username)
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
	rows, err := db.Query(`SELECT banned_at, unbanned_at FROM ban_history WHERE username = ? ORDER BY banned_at ASC`, username)
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
	_, err := db.Exec(`DELETE FROM messages`)
	return err
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
	_, err = db.Exec("PRAGMA wal_checkpoint(FULL);")
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
	_, err = db.Exec(fmt.Sprintf("VACUUM INTO '%s';", backupPath))
	if err != nil {
		return "", fmt.Errorf("failed to create database backup: %v", err)
	}

	return backupFilename, nil
}

// GetDatabaseStats returns statistics about the database
func GetDatabaseStats(db *sql.DB) (string, error) {
	var stats strings.Builder

	// Count messages
	var messageCount int
	err := db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&messageCount)
	if err != nil {
		return "", fmt.Errorf("failed to count messages: %v", err)
	}

	// Count unique users
	var userCount int
	err = db.QueryRow("SELECT COUNT(DISTINCT sender) FROM messages WHERE sender != 'System'").Scan(&userCount)
	if err != nil {
		return "", fmt.Errorf("failed to count users: %v", err)
	}

	// Get oldest and newest message dates
	var oldestDate, newestDate sql.NullString
	err = db.QueryRow("SELECT MIN(created_at), MAX(created_at) FROM messages").Scan(&oldestDate, &newestDate)
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
	sort.Strings(usernames) // Sort alphabetically
	userList := UserList{Users: usernames}
	payload, _ := json.Marshal(userList)
	msg := WSMessage{Type: "userlist", Data: payload}
	for client := range h.clients {
		client.send <- msg
	}
	h.clientsMutex.RUnlock()
}

type adminAuth struct {
	admins   map[string]struct{}
	adminKey string
}

// validateUsernameHandler validates username format (same logic as client.go validateUsername)
func validateUsernameHandler(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if len(username) > 32 {
		return fmt.Errorf("username too long (max 32 characters)")
	}
	// Allow letters, numbers, underscores, hyphens, and periods
	for _, char := range username {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-' || char == '.') {
			return fmt.Errorf("username contains invalid characters (only letters, numbers, _, -, . allowed)")
		}
	}
	// Prevent usernames that could be confused with system messages or commands
	if strings.HasPrefix(username, ":") || strings.HasPrefix(username, ".") {
		return fmt.Errorf("username cannot start with ':' or '.'")
	}
	// Prevent path traversal attempts
	if strings.Contains(username, "..") {
		return fmt.Errorf("username cannot contain '..'")
	}
	return nil
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
			if err := conn.WriteMessage(websocket.CloseMessage, []byte("Invalid handshake")); err != nil {
				log.Printf("WriteMessage error: %v", err)
			}
			conn.Close()
			return
		}
		username := strings.TrimSpace(hs.Username)
		if username == "" {
			if err := conn.WriteMessage(websocket.CloseMessage, []byte("Username required")); err != nil {
				log.Printf("WriteMessage error: %v", err)
			}
			conn.Close()
			return
		}

		// Validate username format
		if err := validateUsernameHandler(username); err != nil {
			SecurityLogger.Warn("Invalid username attempt", map[string]interface{}{
				"username": username,
				"error":    err.Error(),
				"ip":       getClientIP(r),
			})
			if err := conn.WriteMessage(websocket.CloseMessage, []byte("Invalid username: "+err.Error())); err != nil {
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
				if err := conn.WriteMessage(websocket.CloseMessage, []byte("Username not allowed on this server")); err != nil {
					log.Printf("WriteMessage error: %v", err)
				}
				conn.Close()
				return
			}
		}
		isAdmin := false
		if hs.Admin {
			if _, ok := auth.admins[lu]; !ok {
				if err := conn.WriteMessage(websocket.CloseMessage, []byte("Not an admin user")); err != nil {
					log.Printf("WriteMessage error: %v", err)
				}
				conn.Close()
				return
			}
			if hs.AdminKey != auth.adminKey {
				// Send auth_failed message before closing
				failMsg, _ := json.Marshal(map[string]string{"reason": "invalid admin key"})
				if err := conn.WriteJSON(WSMessage{Type: "auth_failed", Data: failMsg}); err != nil {
					log.Printf("WriteMessage error: %v", err)
				}
				conn.Close()
				return
			}
			isAdmin = true
		}

		// Extract IP address
		ipAddr := getClientIP(r)

		// Check for duplicate username
		hub.clientsMutex.RLock()
		isDuplicate := false
		for client := range hub.clients {
			if strings.EqualFold(client.username, username) {
				log.Printf("Duplicate username attempt: '%s' (IP: %s) - username already in use by IP: %s", username, ipAddr, client.ipAddr)
				isDuplicate = true
				break
			}
		}
		hub.clientsMutex.RUnlock()
		if isDuplicate {
			if err := conn.WriteMessage(websocket.CloseMessage, []byte("Username already taken - please choose a different username")); err != nil {
				log.Printf("WriteMessage error: %v", err)
			}
			conn.Close()
			return
		}

		// Check if user is banned
		if hub.IsUserBanned(username) {
			log.Printf("Banned user '%s' (IP: %s) attempted to connect", username, ipAddr)
			if err := conn.WriteMessage(websocket.CloseMessage, []byte("You are banned from this server")); err != nil {
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

		// Send personalized recent messages to new client
		msgs, _ := GetRecentMessagesForUser(db, username, 50, banGapsHistory)
		for _, msg := range msgs {
			client.send <- msg
		}
		hub.broadcastUserList()

		// Start read/write pumps
		go client.writePump()
		go client.readPump()
	}
}
