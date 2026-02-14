package server

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"

	"github.com/gorilla/websocket"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10 // send pings at 90% of pongWait
)

type Client struct {
	hub                  *Hub
	conn                 *websocket.Conn
	send                 chan interface{}
	db                   *sql.DB
	username             string
	isAdmin              bool
	ipAddr               string // Store IP address for logging and ban enforcement
	pluginCommandHandler *PluginCommandHandler
	maxFileBytes         int64
	dbPath               string // Store database path for backup operations
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	// Allow up to configured max file size (+ small overhead for JSON framing)
	limit := int64(1024*1024) + 512
	if c.maxFileBytes > 0 {
		limit = c.maxFileBytes + 512
	}
	c.conn.SetReadLimit(limit)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("SetReadDeadline error: %v", err)
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Printf("SetReadDeadline error: %v", err)
		}
		return nil
	})
	for {
		var msg shared.Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseAbnormalClosure) {
				log.Printf("Client %s disconnected unexpectedly: %v", c.username, err)
			} else {
				log.Printf("Client %s disconnected normally", c.username)
			}
			break
		}
		if msg.Type == shared.FileMessageType && msg.File != nil {
			// File message: enforce configured limit
			maxBytes := c.maxFileBytes
			if maxBytes <= 0 {
				maxBytes = 1024 * 1024
			}
			if msg.File.Size > maxBytes {
				log.Printf("Rejected file from %s: too large (%d bytes)", c.username, msg.File.Size)
				continue
			}
			// Broadcast file message, do not store in DB
			msg.CreatedAt = time.Now()
			c.hub.broadcast <- msg
			continue
		}
		// Handle commands (both plugin and admin commands)
		if strings.HasPrefix(msg.Content, ":") || msg.Type == shared.AdminCommandType {
			AdminLogger.Info("Command received", map[string]interface{}{
				"user":    c.username,
				"command": msg.Content,
				"admin":   c.isAdmin,
				"type":    msg.Type,
			})
			// Let handleCommand process both plugin and admin commands
			// It will check permissions for each command individually
			c.handleCommand(msg.Content)
			continue // Don't insert commands as normal messages
		}
		msg.CreatedAt = time.Now()
		if msg.Type == "" || msg.Type == shared.TextMessage {
			if err := InsertMessage(c.db, msg); err != nil {
				log.Printf("Failed to persist message from %s: %v", c.username, err)
			}
		}
		c.hub.broadcast <- msg
	}
}

// validateUsername ensures usernames are safe and cannot cause injection attacks
func validateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if len(username) > 32 {
		return fmt.Errorf("username too long (max 32 characters)")
	}
	// Allow letters, numbers, underscores, hyphens, and periods
	// This prevents log injection, command injection, and other issues
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

// parseCommandWithQuotes parses a command string, respecting quoted arguments
func parseCommandWithQuotes(command string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	escapeNext := false

	for _, char := range command {
		if escapeNext {
			current.WriteRune(char)
			escapeNext = false
			continue
		}

		if char == '\\' {
			escapeNext = true
			continue
		}

		if char == '"' {
			inQuotes = !inQuotes
			continue
		}

		if char == ' ' && !inQuotes {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(char)
		}
	}

	// Add the last part
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// handleCommand processes both plugin commands and built-in admin commands
func (c *Client) handleCommand(command string) {
	// Parse command with proper quote handling
	parts := parseCommandWithQuotes(command)
	if len(parts) == 0 {
		return
	}

	// First, try to handle plugin commands (these have their own permission checks)
	if c.pluginCommandHandler != nil {
		cmd := strings.TrimPrefix(parts[0], ":")
		args := parts[1:]

		AdminLogger.Info("Trying plugin command", map[string]interface{}{
			"user":    c.username,
			"command": cmd,
			"args":    args,
		})
		response, err := c.pluginCommandHandler.HandlePluginCommand(cmd, args, c.isAdmin)
		if err == nil {
			// Log plugin operations at INFO level so they show in admin panel
			AdminLogger.Info("Plugin command executed", map[string]interface{}{
				"user":     c.username,
				"command":  cmd,
				"args":     args,
				"response": response,
			})
			// Plugin command was handled successfully
			AdminLogger.Debug("Plugin command handled successfully", map[string]interface{}{
				"user":     c.username,
				"response": response,
			})
			c.send <- shared.Message{
				Sender:    "System",
				Content:   response,
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
			return
		} else {
			AdminLogger.Debug("Plugin command failed", map[string]interface{}{
				"user":  c.username,
				"error": err.Error(),
			})
		}
	} else {
		AdminLogger.Debug("No plugin command handler available", map[string]interface{}{
			"user": c.username,
		})
	}

	// Fall back to built-in admin commands (these require admin privileges)
	// Check admin status for built-in commands
	if !c.isAdmin {
		SecurityLogger.Warn("Unauthorized admin command attempt", map[string]interface{}{
			"user":    c.username,
			"command": parts[0],
		})
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "This command requires admin privileges",
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
		return
	}
	switch parts[0] {
	case ":cleardb":
		log.Printf("[ADMIN] Clearing message database via WebSocket by %s...", c.username)
		err := ClearMessages(c.db)
		if err != nil {
			log.Printf("Failed to clear DB: %v", err)
		} else {
			log.Printf("Message DB cleared by %s", c.username)
			c.hub.broadcast <- shared.Message{
				Sender:    "System",
				Content:   "Chat history cleared by admin.",
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
		}

	case ":kick":
		if len(parts) < 2 {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Usage: :kick <username>",
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
			return
		}
		targetUsername := parts[1]
		if err := validateUsername(targetUsername); err != nil {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Invalid username: " + err.Error(),
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
			return
		}
		c.hub.KickUser(targetUsername, c.username)
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "User '" + targetUsername + "' has been kicked (24 hour temporary ban).",
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}

	case ":ban":
		if len(parts) < 2 {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Usage: :ban <username>",
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
			return
		}
		targetUsername := parts[1]
		if err := validateUsername(targetUsername); err != nil {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Invalid username: " + err.Error(),
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
			return
		}
		c.hub.BanUser(targetUsername, c.username)
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "User '" + targetUsername + "' has been permanently banned.",
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}

	case ":unban":
		if len(parts) < 2 {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Usage: :unban <username>",
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
			return
		}
		targetUsername := parts[1]
		if err := validateUsername(targetUsername); err != nil {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Invalid username: " + err.Error(),
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
			return
		}
		unbanned := c.hub.UnbanUser(targetUsername, c.username)
		if unbanned {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "User '" + targetUsername + "' has been unbanned.",
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
		} else {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "User '" + targetUsername + "' was not found in the ban list.",
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
		}

	case ":allow":
		if len(parts) < 2 {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Usage: :allow <username>",
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
			return
		}
		targetUsername := parts[1]
		if err := validateUsername(targetUsername); err != nil {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Invalid username: " + err.Error(),
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
			return
		}
		allowed := c.hub.AllowUser(targetUsername, c.username)
		if allowed {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "User '" + targetUsername + "' has been allowed back (kick override).",
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
		} else {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "User '" + targetUsername + "' was not found in the kick list.",
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
		}

	case ":cleanup":
		log.Printf("[ADMIN] Manual stale connection cleanup initiated by %s", c.username)
		c.hub.CleanupStaleConnections()
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "Stale connection cleanup completed.",
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}

	case ":forcedisconnect":
		if len(parts) < 2 {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Usage: :forcedisconnect <username>",
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
			return
		}
		targetUsername := parts[1]
		if err := validateUsername(targetUsername); err != nil {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Invalid username: " + err.Error(),
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
			return
		}
		disconnected := c.hub.ForceDisconnectUser(targetUsername, c.username)
		if disconnected {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "User '" + targetUsername + "' has been forcibly disconnected.",
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
		} else {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "User '" + targetUsername + "' was not found in active connections.",
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
		}

	case ":backup":
		log.Printf("[ADMIN] Database backup requested by %s", c.username)
		// Use the configured database path
		backupFilename, err := BackupDatabase(c.dbPath)
		if err != nil {
			log.Printf("Failed to backup database: %v", err)
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Database backup failed: " + err.Error(),
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
		} else {
			log.Printf("Database backup created: %s", backupFilename)
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Database backup created: " + backupFilename,
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
		}

	case ":stats":
		log.Printf("[ADMIN] Database stats requested by %s", c.username)
		stats, err := GetDatabaseStats(c.db)
		if err != nil {
			log.Printf("Failed to get database stats: %v", err)
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Failed to get database stats: " + err.Error(),
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
		} else {
			c.send <- shared.Message{
				Sender:    "System",
				Content:   stats,
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
		}

	default:
		log.Printf("[ADMIN] Unknown admin command by %s: %s", c.username, command)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					if !strings.Contains(err.Error(), "use of closed network connection") {
						log.Printf("WriteMessage error: %v", err)
					}
				}
				return
			}
			switch v := msg.(type) {
			case shared.Message:
				err := c.conn.WriteJSON(v)
				if err != nil {
					if !strings.Contains(err.Error(), "use of closed network connection") {
						log.Printf("Failed to send message to %s: %v", c.username, err)
					}
					return
				}
			case WSMessage:
				err := c.conn.WriteJSON(v)
				if err != nil {
					if !strings.Contains(err.Error(), "use of closed network connection") {
						log.Printf("Failed to send system message to %s: %v", c.username, err)
					}
					return
				}
			default:
				log.Printf("Unknown message type for client %s", c.username)
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") {
					log.Printf("Failed to send ping to %s: %v", c.username, err)
				}
				return
			}
		}
	}
}
