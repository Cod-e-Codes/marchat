package server

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Cod-e-Codes/marchat/plugin/manager"
	"github.com/Cod-e-Codes/marchat/shared"
)

// Sentinel errors for KickUser / BanUser so callers can map clear replies
// and claim success only when err == nil.
var (
	// ErrAdminSelfTarget is returned when an admin tries to kick or ban themselves.
	ErrAdminSelfTarget = errors.New("cannot kick or ban yourself")
	// ErrKickPermanentlyBanned is returned when KickUser targets a permanently banned user.
	ErrKickPermanentlyBanned = errors.New("cannot kick a permanently banned user")
	// ErrKickNotConnected is returned when KickUser targets a user with no active connection.
	ErrKickNotConnected = errors.New("user is not connected")
)

type Hub struct {
	clients           map[*Client]bool
	usernames         map[string]struct{} // reserved names (handshake), lowercased
	clientsByUsername map[string]*Client  // connected clients by lowercased username
	clientsMutex      sync.RWMutex
	broadcast         chan interface{}
	register          chan *Client
	unregister        chan *Client

	// Ban management
	bans      map[string]struct{}  // username -> permanently banned (no expiry)
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

	channels     map[string]map[*Client]bool
	channelMutex sync.RWMutex
}

func NewHub(pluginDir, dataDir, registryURL string, db *sql.DB) (*Hub, error) {
	pluginManager := manager.NewPluginManager(pluginDir, dataDir, registryURL)
	pluginCommandHandler := NewPluginCommandHandler(pluginManager)

	h := &Hub{
		clients:              make(map[*Client]bool),
		usernames:            make(map[string]struct{}),
		clientsByUsername:    make(map[string]*Client),
		broadcast:            make(chan interface{}),
		register:             make(chan *Client),
		unregister:           make(chan *Client),
		bans:                 make(map[string]struct{}),
		tempKicks:            make(map[string]time.Time),
		pluginManager:        pluginManager,
		pluginCommandHandler: pluginCommandHandler,
		db:                   db,
		channels:             make(map[string]map[*Client]bool),
	}
	if db != nil {
		if err := h.loadModerationState(); err != nil {
			return nil, err
		}
	}
	return h, nil
}

// loadModerationState restores active bans and unexpired temp kicks from ban_history.
// Open rows (unbanned_at IS NULL) with NULL expires_at are permanent bans.
// Open rows with expires_at in the future are temp kicks; expired rows are skipped.
func (h *Hub) loadModerationState() error {
	rows, err := dbQuery(h.db, `
		SELECT username, expires_at FROM ban_history
		WHERE unbanned_at IS NULL`)
	if err != nil {
		return fmt.Errorf("load moderation state: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	h.banMutex.Lock()
	defer h.banMutex.Unlock()

	for rows.Next() {
		var username string
		var expiresAt sql.NullTime
		if err := rows.Scan(&username, &expiresAt); err != nil {
			return fmt.Errorf("scan moderation row: %w", err)
		}
		lower := strings.ToLower(username)
		if !expiresAt.Valid {
			// Permanent ban (or pre-upgrade open row treated as permanent).
			h.bans[lower] = struct{}{}
			continue
		}
		if now.Before(expiresAt.Time) {
			h.tempKicks[lower] = expiresAt.Time
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate moderation rows: %w", err)
	}
	return nil
}

func (h *Hub) TryReserveUsername(username string) bool {
	lowerUsername := strings.ToLower(username)

	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()

	if _, exists := h.usernames[lowerUsername]; exists {
		return false
	}
	h.usernames[lowerUsername] = struct{}{}
	return true
}

func (h *Hub) ReleaseUsername(username string) {
	lowerUsername := strings.ToLower(username)

	h.clientsMutex.Lock()
	delete(h.usernames, lowerUsername)
	h.clientsMutex.Unlock()
}

// BanUser adds a user to the permanent ban list.
// The ban state is recorded under banMutex, then the lock is released before
// kicking the connected client so that a blocked send channel cannot hold
// banMutex and stall all other ban/kick callers.
// Returns ErrAdminSelfTarget when username matches adminUsername (case-insensitive).
// Offline bans are still allowed when the target is a different user.
func (h *Hub) BanUser(username string, adminUsername string) error {
	if strings.EqualFold(username, adminUsername) {
		return ErrAdminSelfTarget
	}

	h.banMutex.Lock()

	lowerUsername := strings.ToLower(username)

	// Remove from temporary kicks if present
	delete(h.tempKicks, lowerUsername)

	// Add to permanent bans (no expiry)
	h.bans[lowerUsername] = struct{}{}
	AdminLogger.Info("User permanently banned", map[string]interface{}{
		"banned_user": username,
		"admin":       adminUsername,
	})

	// Record ban event in database (NULL expires_at = permanent)
	if h.getDB() != nil {
		err := recordBanEvent(h.getDB(), lowerUsername, adminUsername, nil)
		if err != nil {
			log.Printf("Warning: failed to record ban event for user %s: %v", username, err)
		}
	}

	// Clear per-user last_seen bookkeeping on ban (ban gaps use ban_history, not this table).
	if h.getDB() != nil {
		err := clearUserMessageState(h.getDB(), lowerUsername)
		if err != nil {
			log.Printf("Warning: failed to clear message state for banned user %s: %v", username, err)
		}
	}

	h.banMutex.Unlock()

	h.kickUser(username, "You have been permanently banned by an administrator")
	return nil
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

		// Clear per-user last_seen bookkeeping on unban.
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

	// Check permanent bans (no expiry)
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

// disconnectClient forcibly disconnects a connected client.
// Uses a non-blocking send so the caller never stalls on a full channel.
func (h *Hub) disconnectClient(target *Client, reason string) {
	if target == nil {
		return
	}

	username := target.username
	log.Printf("[ADMIN] Kicking user '%s' (IP: %s) - Reason: %s", username, target.ipAddr, reason)

	kickMsg := shared.Message{
		Sender:    "System",
		Content:   "You have been kicked by an administrator: " + reason,
		CreatedAt: time.Now(),
		Type:      shared.TextMessage,
	}
	select {
	case target.send <- kickMsg:
	default:
		log.Printf("[ADMIN] Could not deliver kick message to %s (send buffer full)", username)
	}

	if target.conn != nil {
		target.conn.Close()
	}
}

// clientByUsernameLocked returns the connected client for a lowercased username.
// Caller must hold clientsMutex (read or write).
func (h *Hub) clientByUsernameLocked(lowerUsername string) *Client {
	return h.clientsByUsername[lowerUsername]
}

// removeClientLocked removes a client from clients and clientsByUsername.
// Caller must hold clientsMutex for write.
func (h *Hub) removeClientLocked(client *Client) {
	delete(h.clients, client)
	if client != nil && client.username != "" {
		lower := strings.ToLower(client.username)
		if h.clientsByUsername[lower] == client {
			delete(h.clientsByUsername, lower)
		}
	}
}

// kickUser forcibly disconnects a user by username.
func (h *Hub) kickUser(username string, reason string) {
	h.clientsMutex.RLock()
	target := h.clientByUsernameLocked(strings.ToLower(username))
	h.clientsMutex.RUnlock()

	if target == nil {
		log.Printf("[ADMIN] Kick attempt for '%s' - user not found", username)
		return
	}

	h.disconnectClient(target, reason)
}

// KickUser disconnects a connected user and temporarily bans them for 24 hours.
// The target must have an active WebSocket connection; offline users are not
// kicked or temp-banned (use BanUser for offline moderation).
// Like BanUser, banMutex is released before the disconnect to avoid holding the
// lock across a potentially blocking channel send.
// Returns ErrAdminSelfTarget when username matches adminUsername (case-insensitive),
// checked before any tempKicks write. Returns ErrKickNotConnected when the user is
// not connected. Returns ErrKickPermanentlyBanned when the target is already
// permanently banned.
func (h *Hub) KickUser(username string, adminUsername string) error {
	if strings.EqualFold(username, adminUsername) {
		return ErrAdminSelfTarget
	}

	h.clientsMutex.RLock()
	target := h.clientByUsernameLocked(strings.ToLower(username))
	h.clientsMutex.RUnlock()

	if target == nil {
		return ErrKickNotConnected
	}

	h.banMutex.Lock()

	lowerUsername := strings.ToLower(username)

	// Don't override permanent bans with temporary kicks
	if _, isPermanentlyBanned := h.bans[lowerUsername]; isPermanentlyBanned {
		h.banMutex.Unlock()
		log.Printf("[ADMIN] Cannot kick '%s' - user is permanently banned", username)
		return ErrKickPermanentlyBanned
	}

	// Add to temporary kicks for 24 hours
	kickExpiry := time.Now().Add(24 * time.Hour)
	h.tempKicks[lowerUsername] = kickExpiry
	AdminLogger.Info("User kicked", map[string]interface{}{
		"kicked_user": username,
		"admin":       adminUsername,
		"until":       kickExpiry.Format("2006-01-02 15:04:05"),
	})

	// Record kick event in database with 24h expiry
	if h.getDB() != nil {
		exp := kickExpiry
		err := recordBanEvent(h.getDB(), lowerUsername, adminUsername, &exp)
		if err != nil {
			log.Printf("Warning: failed to record kick event for user %s: %v", username, err)
		}
	}

	// Clear per-user last_seen bookkeeping on kick (ban gaps use ban_history, not this table).
	if h.getDB() != nil {
		err := clearUserMessageState(h.getDB(), lowerUsername)
		if err != nil {
			log.Printf("Warning: failed to clear message state for kicked user %s: %v", username, err)
		}
	}

	h.banMutex.Unlock()

	h.disconnectClient(target, "You have been kicked by an administrator (24 hour temporary ban)")
	return nil
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

// CleanupExpiredBans removes expired temporary kicks from the lists.
// Permanent bans have no expiry and are never cleared here.
func (h *Hub) CleanupExpiredBans() {
	h.banMutex.Lock()
	defer h.banMutex.Unlock()

	now := time.Now()

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
		// Try to ping the client to check if connection is alive (serialized with writePump)
		if err := client.PingConn(); err != nil {
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
			h.removeClientLocked(client)
			delete(h.usernames, strings.ToLower(client.username))
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
	target := h.clientByUsernameLocked(strings.ToLower(username))
	if target == nil {
		h.clientsMutex.Unlock()
		log.Printf("[ADMIN] Force disconnect attempt for '%s' by '%s' - user not found", username, adminUsername)
		return false
	}

	log.Printf("[ADMIN] Force disconnecting user '%s' (IP: %s) by admin '%s'", username, target.ipAddr, adminUsername)

	h.removeClientLocked(target)
	delete(h.usernames, strings.ToLower(target.username))
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
			h.broadcast <- ConvertPluginMessage(msg)
		}
	}()

	for {
		select {
		case client := <-h.register:
			h.clientsMutex.Lock()
			h.clients[client] = true
			if client.username != "" {
				h.clientsByUsername[strings.ToLower(client.username)] = client
			}
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
			h.joinChannel(client, "general")
			if h.pluginCommandHandler != nil {
				h.pluginCommandHandler.UpdateUserListForPlugins(h.getConnectedUsernames())
			}
		case client := <-h.unregister:
			h.clientsMutex.Lock()
			if _, ok := h.clients[client]; ok {
				h.removeClientLocked(client)
				delete(h.usernames, strings.ToLower(client.username))
				// Intentionally do not close client.send here.
				// Closing send while readPump is still processing can trigger send-on-closed-channel panics.
				// Connection close is the teardown signal; writePump exits on failed writes/pings.
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
			h.channelMutex.Lock()
			for ch, clients := range h.channels {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.channels, ch)
				}
			}
			h.channelMutex.Unlock()
			h.broadcastUserList()
			if h.pluginCommandHandler != nil {
				h.pluginCommandHandler.UpdateUserListForPlugins(h.getConnectedUsernames())
			}
		case message := <-h.broadcast:
			h.clientsMutex.Lock()
			if sharedMsg, ok := message.(shared.Message); ok && sharedMsg.Channel != "" && sharedMsg.Sender != "System" {
				h.channelMutex.RLock()
				channelClients := h.channels[sharedMsg.Channel]
				for client := range h.clients {
					if channelClients[client] {
						select {
						case client.send <- message:
						default:
							log.Printf("Dropping client %s due to full send channel\n", client.username)
							h.removeClientLocked(client)
							delete(h.usernames, strings.ToLower(client.username))
							// Fail-fast backpressure handling: drop slow client and close socket.
							client.conn.Close()
						}
					}
				}
				h.channelMutex.RUnlock()
			} else {
				for client := range h.clients {
					select {
					case client.send <- message:
					default:
						log.Printf("Dropping client %s due to full send channel\n", client.username)
						h.removeClientLocked(client)
						delete(h.usernames, strings.ToLower(client.username))
						// Fail-fast backpressure handling: drop slow client and close socket.
						client.conn.Close()
					}
				}
			}
			h.clientsMutex.Unlock()

			if sharedMsg, ok := message.(shared.Message); ok && sharedMsg.Type == shared.TextMessage {
				if h.pluginCommandHandler != nil {
					// Never block the hub on plugin IPC; fan-out runs in its own goroutine.
					msgCopy := sharedMsg
					go h.pluginCommandHandler.SendMessageToPlugins(msgCopy)
				}
			}
		}
	}
}

func (h *Hub) broadcastDM(msg shared.Message) {
	h.clientsMutex.RLock()
	defer h.clientsMutex.RUnlock()
	seen := make(map[*Client]struct{}, 2)
	for _, name := range []string{msg.Sender, msg.Recipient} {
		if name == "" {
			continue
		}
		client := h.clientByUsernameLocked(strings.ToLower(name))
		if client == nil {
			continue
		}
		if _, ok := seen[client]; ok {
			continue
		}
		seen[client] = struct{}{}
		select {
		case client.send <- msg:
		default:
			log.Printf("Dropping DM for client %s due to full send channel", client.username)
		}
	}
}

// getConnectedUsernames returns the usernames of all connected clients.
// Caller must NOT hold clientsMutex.
func (h *Hub) getConnectedUsernames() []string {
	h.clientsMutex.RLock()
	defer h.clientsMutex.RUnlock()

	var usernames []string
	for client := range h.clients {
		if client.username != "" {
			usernames = append(usernames, client.username)
		}
	}
	return usernames
}

// getDB returns the database reference
func (h *Hub) getDB() *sql.DB {
	return h.db
}

func (h *Hub) joinChannel(client *Client, channel string) {
	h.channelMutex.Lock()
	defer h.channelMutex.Unlock()
	if h.channels[channel] == nil {
		h.channels[channel] = make(map[*Client]bool)
	}
	h.channels[channel][client] = true
}

func (h *Hub) leaveChannel(client *Client, channel string) {
	h.channelMutex.Lock()
	defer h.channelMutex.Unlock()
	if h.channels[channel] != nil {
		delete(h.channels[channel], client)
		if len(h.channels[channel]) == 0 {
			delete(h.channels, channel)
		}
	}
}

func (h *Hub) getClientChannel(client *Client) string {
	h.channelMutex.RLock()
	defer h.channelMutex.RUnlock()
	for channel, clients := range h.channels {
		if clients[client] {
			return channel
		}
	}
	return "general"
}

func (h *Hub) getChannelUsers(channel string) []*Client {
	h.channelMutex.RLock()
	defer h.channelMutex.RUnlock()
	var users []*Client
	for c := range h.channels[channel] {
		users = append(users, c)
	}
	return users
}

func (h *Hub) listChannels() []string {
	h.channelMutex.RLock()
	defer h.channelMutex.RUnlock()
	var channels []string
	for ch := range h.channels {
		channels = append(channels, ch)
	}
	if len(channels) == 0 {
		channels = append(channels, "general")
	}
	return channels
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
