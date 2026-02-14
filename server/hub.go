package server

import (
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Cod-e-Codes/marchat/plugin/manager"
	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/gorilla/websocket"
)

type Hub struct {
	clients      map[*Client]bool
	clientsMutex sync.RWMutex
	broadcast    chan interface{}
	register     chan *Client
	unregister   chan *Client

	// Ban management
	bans      map[string]time.Time // username -> expiry time (permanent bans use far future time)
	tempKicks map[string]time.Time // username -> kick expiry time (24h temporary)
	banMutex  sync.RWMutex

	// Metrics tracking
	totalConnections int
	totalDisconnects int
	metricsMutex     sync.RWMutex

	// Plugin management
	pluginManager        *manager.PluginManager
	pluginCommandHandler *PluginCommandHandler

	// Database reference for message state management
	db *sql.DB
}

func NewHub(pluginDir, dataDir, registryURL string, db *sql.DB) *Hub {
	pluginManager := manager.NewPluginManager(pluginDir, dataDir, registryURL)
	pluginCommandHandler := NewPluginCommandHandler(pluginManager)

	return &Hub{
		clients:              make(map[*Client]bool),
		broadcast:            make(chan interface{}),
		register:             make(chan *Client),
		unregister:           make(chan *Client),
		bans:                 make(map[string]time.Time),
		tempKicks:            make(map[string]time.Time),
		pluginManager:        pluginManager,
		pluginCommandHandler: pluginCommandHandler,
		db:                   db,
	}
}

// BanUser adds a user to the permanent ban list
func (h *Hub) BanUser(username string, adminUsername string) {
	h.banMutex.Lock()
	defer h.banMutex.Unlock()

	lowerUsername := strings.ToLower(username)

	// Remove from temporary kicks if present
	delete(h.tempKicks, lowerUsername)

	// Add to permanent bans (using far future time to indicate permanent)
	permanentBanTime := time.Now().Add(100 * 365 * 24 * time.Hour) // 100 years in the future
	h.bans[lowerUsername] = permanentBanTime
	AdminLogger.Info("User permanently banned", map[string]interface{}{
		"banned_user": username,
		"admin":       adminUsername,
	})

	// Record ban event in database
	if h.getDB() != nil {
		err := recordBanEvent(h.getDB(), lowerUsername, adminUsername)
		if err != nil {
			log.Printf("Warning: failed to record ban event for user %s: %v", username, err)
		}
	}

	// Clear user's message state to ensure fresh history on unban
	if h.getDB() != nil {
		err := clearUserMessageState(h.getDB(), lowerUsername)
		if err != nil {
			log.Printf("Warning: failed to clear message state for banned user %s: %v", username, err)
		}
	}

	// Kick the user if they're currently connected
	h.kickUser(username, "You have been permanently banned by an administrator")
}

// UnbanUser removes a user from the ban list
func (h *Hub) UnbanUser(username string, adminUsername string) bool {
	h.banMutex.Lock()
	defer h.banMutex.Unlock()

	lowerUsername := strings.ToLower(username)
	if _, exists := h.bans[lowerUsername]; exists {
		delete(h.bans, lowerUsername)
		AdminLogger.Info("User unbanned", map[string]interface{}{
			"unbanned_user": username,
			"admin":         adminUsername,
		})

		// Record unban event in database
		if h.getDB() != nil {
			err := recordUnbanEvent(h.getDB(), lowerUsername)
			if err != nil {
				log.Printf("Warning: failed to record unban event for user %s: %v", username, err)
			}
		}

		// Clear user's message state to ensure clean slate on reconnection
		if h.getDB() != nil {
			err := clearUserMessageState(h.getDB(), lowerUsername)
			if err != nil {
				log.Printf("Warning: failed to clear message state for unbanned user %s: %v", username, err)
			}
		}

		return true
	}
	log.Printf("[ADMIN] Unban attempt for '%s' by '%s' - user not found in ban list", username, adminUsername)
	return false
}

// IsUserBanned checks if a user is currently banned or kicked
func (h *Hub) IsUserBanned(username string) bool {
	h.banMutex.RLock()
	defer h.banMutex.RUnlock()

	lowerUsername := strings.ToLower(username)

	// Check permanent bans (these don't expire automatically)
	if _, exists := h.bans[lowerUsername]; exists {
		return true
	}

	// Check temporary kicks
	if kickTime, exists := h.tempKicks[lowerUsername]; exists {
		if time.Now().Before(kickTime) {
			return true
		}
		// Kick has expired, but don't remove it here - let CleanupExpiredBans handle it
		// This prevents race conditions with the :allow command
	}

	return false
}

// kickUser forcibly disconnects a user by username
func (h *Hub) kickUser(username string, reason string) {
	h.clientsMutex.RLock()
	var target *Client
	for client := range h.clients {
		if strings.EqualFold(client.username, username) {
			target = client
			break
		}
	}
	h.clientsMutex.RUnlock()

	if target == nil {
		log.Printf("[ADMIN] Kick attempt for '%s' - user not found", username)
		return
	}

	log.Printf("[ADMIN] Kicking user '%s' (IP: %s) - Reason: %s", username, target.ipAddr, reason)

	// Send kick message to the user
	kickMsg := shared.Message{
		Sender:    "System",
		Content:   "You have been kicked by an administrator: " + reason,
		CreatedAt: time.Now(),
		Type:      shared.TextMessage,
	}
	target.send <- kickMsg

	// Close the connection
	target.conn.Close()
}

// KickUser temporarily bans a user for 24 hours
func (h *Hub) KickUser(username string, adminUsername string) {
	h.banMutex.Lock()
	defer h.banMutex.Unlock()

	lowerUsername := strings.ToLower(username)

	// Don't override permanent bans with temporary kicks
	if _, isPermanentlyBanned := h.bans[lowerUsername]; isPermanentlyBanned {
		log.Printf("[ADMIN] Cannot kick '%s' - user is permanently banned", username)
		return
	}

	// Add to temporary kicks for 24 hours
	kickExpiry := time.Now().Add(24 * time.Hour)
	h.tempKicks[lowerUsername] = kickExpiry
	AdminLogger.Info("User kicked", map[string]interface{}{
		"kicked_user": username,
		"admin":       adminUsername,
		"until":       kickExpiry.Format("2006-01-02 15:04:05"),
	})

	// Record kick event in database (reuse ban event structure)
	if h.getDB() != nil {
		err := recordBanEvent(h.getDB(), lowerUsername, adminUsername)
		if err != nil {
			log.Printf("Warning: failed to record kick event for user %s: %v", username, err)
		}
	}

	// Clear user's message state
	if h.getDB() != nil {
		err := clearUserMessageState(h.getDB(), lowerUsername)
		if err != nil {
			log.Printf("Warning: failed to clear message state for kicked user %s: %v", username, err)
		}
	}

	// Disconnect the user if they're currently connected
	h.kickUser(username, "You have been kicked by an administrator (24 hour temporary ban)")
}

// AllowUser removes a user from temporary kick list (override early)
func (h *Hub) AllowUser(username string, adminUsername string) bool {
	h.banMutex.Lock()
	defer h.banMutex.Unlock()

	lowerUsername := strings.ToLower(username)

	// Check if user is in temporary kick list
	if _, exists := h.tempKicks[lowerUsername]; exists {
		delete(h.tempKicks, lowerUsername)
		log.Printf("[ADMIN] User '%s' allowed back by '%s' (kick override)", username, adminUsername)

		// Record unban event in database
		if h.getDB() != nil {
			err := recordUnbanEvent(h.getDB(), lowerUsername)
			if err != nil {
				log.Printf("Warning: failed to record allow event for user %s: %v", username, err)
			}
		}

		return true
	}

	log.Printf("[ADMIN] Allow attempt for '%s' by '%s' - user not found in kick list", username, adminUsername)
	return false
}

// CleanupExpiredBans removes expired bans and kicks from the lists
func (h *Hub) CleanupExpiredBans() {
	h.banMutex.Lock()
	defer h.banMutex.Unlock()

	now := time.Now()

	// Clean up expired permanent bans (shouldn't happen with our 100-year approach, but just in case)
	for username, banTime := range h.bans {
		if now.After(banTime) {
			delete(h.bans, username)
			log.Printf("[SYSTEM] Expired permanent ban removed for user: %s", username)
		}
	}

	// Clean up expired temporary kicks
	for username, kickTime := range h.tempKicks {
		if now.After(kickTime) {
			delete(h.tempKicks, username)
			log.Printf("[SYSTEM] Expired kick removed for user: %s", username)
		}
	}
}

// CleanupStaleConnections removes clients with broken connections
func (h *Hub) CleanupStaleConnections() {
	h.clientsMutex.RLock()
	var staleClients []*Client

	// Check all clients for broken connections
	for client := range h.clients {
		// Try to ping the client to check if connection is alive
		if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Printf("[CLEANUP] Found stale connection for user '%s' (IP: %s): %v", client.username, client.ipAddr, err)
			staleClients = append(staleClients, client)
		}
	}
	h.clientsMutex.RUnlock()

	if len(staleClients) == 0 {
		return
	}

	// Remove stale clients
	h.clientsMutex.Lock()
	for _, client := range staleClients {
		if _, exists := h.clients[client]; exists {
			log.Printf("[CLEANUP] Removing stale connection for user '%s' (IP: %s)", client.username, client.ipAddr)
			delete(h.clients, client)
			close(client.send)
			client.conn.Close()
		}
	}
	h.clientsMutex.Unlock()

	log.Printf("[CLEANUP] Removed %d stale connections", len(staleClients))
	h.broadcastUserList()
}

// ForceDisconnectUser forcibly removes a user from the clients map (admin command for stale connections)
func (h *Hub) ForceDisconnectUser(username string, adminUsername string) bool {
	h.clientsMutex.Lock()
	var target *Client
	for client := range h.clients {
		if strings.EqualFold(client.username, username) {
			target = client
			break
		}
	}
	if target == nil {
		h.clientsMutex.Unlock()
		log.Printf("[ADMIN] Force disconnect attempt for '%s' by '%s' - user not found", username, adminUsername)
		return false
	}

	log.Printf("[ADMIN] Force disconnecting user '%s' (IP: %s) by admin '%s'", username, target.ipAddr, adminUsername)

	// Remove from clients map
	delete(h.clients, target)
	close(target.send)
	h.clientsMutex.Unlock()

	// Close connection outside the lock
	target.conn.Close()

	h.broadcastUserList()
	return true
}

func (h *Hub) Run() {
	HubLogger.Info("Hub started", map[string]interface{}{
		"plugin_manager": h.pluginManager != nil,
	})

	// Start ban cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Hour) // Clean up every hour
		defer ticker.Stop()
		for range ticker.C {
			h.CleanupExpiredBans()
		}
	}()

	// Start stale connection cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // Check for stale connections every 5 minutes
		defer ticker.Stop()
		for range ticker.C {
			h.CleanupStaleConnections()
		}
	}()

	// Start plugin message handler goroutine
	go func() {
		for msg := range h.pluginManager.GetMessageChannel() {
			// Route plugin messages through the broadcast channel for safe access
			sharedMsg := shared.Message{
				Sender:    msg.Sender,
				Content:   msg.Content,
				CreatedAt: msg.CreatedAt,
				Type:      shared.TextMessage,
			}
			h.broadcast <- sharedMsg
		}
	}()

	for {
		select {
		case client := <-h.register:
			h.clientsMutex.Lock()
			h.clients[client] = true
			h.clientsMutex.Unlock()
			HubLogger.Info("Client registered", map[string]interface{}{
				"username": client.username,
				"ip":       client.ipAddr,
			})

			// Update metrics
			h.metricsMutex.Lock()
			h.totalConnections++
			h.metricsMutex.Unlock()

			h.broadcastUserList() // Broadcast after register
		case client := <-h.unregister:
			h.clientsMutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				HubLogger.Info("Client unregistered", map[string]interface{}{
					"username": client.username,
					"ip":       client.ipAddr,
				})

				// Update metrics
				h.metricsMutex.Lock()
				h.totalDisconnects++
				h.metricsMutex.Unlock()
			}
			h.clientsMutex.Unlock()
			h.broadcastUserList()
		case message := <-h.broadcast:
			h.clientsMutex.Lock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					log.Printf("Dropping client %s due to full send channel\n", client.username)
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.clientsMutex.Unlock()
		}
	}
}

// getDB returns the database reference
func (h *Hub) getDB() *sql.DB {
	return h.db
}

// GetPluginManager returns the plugin manager reference
func (h *Hub) GetPluginManager() *manager.PluginManager {
	return h.pluginManager
}

// GetTotalConnections returns the total number of connections since server start
func (h *Hub) GetTotalConnections() int {
	h.metricsMutex.RLock()
	defer h.metricsMutex.RUnlock()
	return h.totalConnections
}

// GetTotalDisconnects returns the total number of disconnections since server start
func (h *Hub) GetTotalDisconnects() int {
	h.metricsMutex.RLock()
	defer h.metricsMutex.RUnlock()
	return h.totalDisconnects
}
