package server

import (
	"database/sql"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
	_ "modernc.org/sqlite"
)

func TestNewHub(t *testing.T) {
	// Create a test database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}

	if hub.clients == nil {
		t.Error("clients map should not be nil")
	}

	if hub.broadcast == nil {
		t.Error("broadcast channel should not be nil")
	}

	if hub.register == nil {
		t.Error("register channel should not be nil")
	}

	if hub.unregister == nil {
		t.Error("unregister channel should not be nil")
	}

	if hub.bans == nil {
		t.Error("bans map should not be nil")
	}

	if hub.tempKicks == nil {
		t.Error("tempKicks map should not be nil")
	}

	if hub.pluginManager == nil {
		t.Error("pluginManager should not be nil")
	}

	if hub.pluginCommandHandler == nil {
		t.Error("pluginCommandHandler should not be nil")
	}

	if hub.db != db {
		t.Error("database reference should be set correctly")
	}
}

func TestHubBanUser(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema for database operations
	CreateSchema(db)

	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	username := "testuser"
	adminUsername := "admin"

	// Test banning a user
	hub.BanUser(username, adminUsername)

	// Check if user is banned
	if !hub.IsUserBanned(username) {
		t.Error("User should be banned")
	}

	// Check case insensitive
	if !hub.IsUserBanned(strings.ToUpper(username)) {
		t.Error("Ban should be case insensitive")
	}

	// Check that ban is permanent (should not expire automatically)
	time.Sleep(100 * time.Millisecond) // Small delay
	if !hub.IsUserBanned(username) {
		t.Error("Permanent ban should not expire")
	}
}

func TestHubUnbanUser(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema for database operations
	CreateSchema(db)

	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	username := "testuser"
	adminUsername := "admin"

	// First ban the user
	hub.BanUser(username, adminUsername)
	if !hub.IsUserBanned(username) {
		t.Error("User should be banned")
	}

	// Now unban the user
	unbanned := hub.UnbanUser(username, adminUsername)
	if !unbanned {
		t.Error("Unban should return true for existing ban")
	}

	// Check if user is unbanned
	if hub.IsUserBanned(username) {
		t.Error("User should not be banned after unban")
	}

	// Test unbanning non-existent user
	unbanned = hub.UnbanUser("nonexistent", adminUsername)
	if unbanned {
		t.Error("Unban should return false for non-existent ban")
	}
}

func TestHubKickUser(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema for database operations
	CreateSchema(db)

	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	username := "testuser"
	adminUsername := "admin"

	// Test kicking a user
	hub.KickUser(username, adminUsername)

	// Check if user is kicked (temporarily banned)
	if !hub.IsUserBanned(username) {
		t.Error("User should be kicked")
	}

	// Check case insensitive
	if !hub.IsUserBanned(strings.ToUpper(username)) {
		t.Error("Kick should be case insensitive")
	}
}

func TestHubAllowUser(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema for database operations
	CreateSchema(db)

	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	username := "testuser"
	adminUsername := "admin"

	// First kick the user
	hub.KickUser(username, adminUsername)
	if !hub.IsUserBanned(username) {
		t.Error("User should be kicked")
	}

	// Now allow the user back
	allowed := hub.AllowUser(username, adminUsername)
	if !allowed {
		t.Error("Allow should return true for existing kick")
	}

	// Check if user is allowed back
	if hub.IsUserBanned(username) {
		t.Error("User should not be banned after allow")
	}

	// Test allowing non-kicked user
	allowed = hub.AllowUser("nonexistent", adminUsername)
	if allowed {
		t.Error("Allow should return false for non-kicked user")
	}
}

func TestHubBanOverridesKick(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema for database operations
	CreateSchema(db)

	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	username := "testuser"
	adminUsername := "admin"

	// First kick the user
	hub.KickUser(username, adminUsername)
	if !hub.IsUserBanned(username) {
		t.Error("User should be kicked")
	}

	// Now ban the user (should override kick)
	hub.BanUser(username, adminUsername)
	if !hub.IsUserBanned(username) {
		t.Error("User should be banned")
	}

	// Try to kick a permanently banned user (should not work)
	hub.KickUser(username, adminUsername)
	if !hub.IsUserBanned(username) {
		t.Error("Permanently banned user should remain banned")
	}
}

func TestHubCleanupExpiredBans(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema for database operations
	CreateSchema(db)

	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	username := "testuser"
	adminUsername := "admin"

	// Kick a user (24 hour temporary ban)
	hub.KickUser(username, adminUsername)
	if !hub.IsUserBanned(username) {
		t.Error("User should be kicked")
	}

	// Manually set the kick time to the past (simulate expired kick)
	hub.banMutex.Lock()
	hub.tempKicks[strings.ToLower(username)] = time.Now().Add(-1 * time.Hour)
	hub.banMutex.Unlock()

	// Run cleanup
	hub.CleanupExpiredBans()

	// User should no longer be banned
	if hub.IsUserBanned(username) {
		t.Error("User should not be banned after cleanup")
	}
}

func TestHubForceDisconnectUser(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	username := "testuser"
	adminUsername := "admin"

	// Test force disconnecting non-existent user
	disconnected := hub.ForceDisconnectUser(username, adminUsername)
	if disconnected {
		t.Error("ForceDisconnectUser should return false for non-existent user")
	}

	// Note: Testing with actual clients would require more complex setup
	// with WebSocket connections, which is beyond the scope of unit tests
}

func TestHubGetPluginManager(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	pluginManager := hub.GetPluginManager()
	if pluginManager == nil {
		t.Error("GetPluginManager should return non-nil plugin manager")
	}

	if pluginManager != hub.pluginManager {
		t.Error("GetPluginManager should return the same plugin manager instance")
	}
}

func TestHubBanCaseInsensitive(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema for database operations
	CreateSchema(db)

	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	username := "TestUser"
	adminUsername := "admin"

	// Ban user with mixed case
	hub.BanUser(username, adminUsername)

	// Test various case combinations
	testCases := []string{
		"testuser",
		"TESTUSER",
		"TestUser",
		"tEsTuSeR",
	}

	for _, testCase := range testCases {
		if !hub.IsUserBanned(testCase) {
			t.Errorf("Ban should be case insensitive for %s", testCase)
		}
	}
}

func TestHubMultipleBansAndKicks(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema for database operations
	CreateSchema(db)

	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	adminUsername := "admin"
	users := []string{"user1", "user2", "user3"}

	// Ban multiple users
	for _, user := range users {
		hub.BanUser(user, adminUsername)
	}

	// Check all users are banned
	for _, user := range users {
		if !hub.IsUserBanned(user) {
			t.Errorf("User %s should be banned", user)
		}
	}

	// Unban one user
	if !hub.UnbanUser("user2", adminUsername) {
		t.Error("Should be able to unban user2")
	}

	if hub.IsUserBanned("user2") {
		t.Error("user2 should not be banned after unban")
	}

	// Other users should still be banned
	if !hub.IsUserBanned("user1") {
		t.Error("user1 should still be banned")
	}

	if !hub.IsUserBanned("user3") {
		t.Error("user3 should still be banned")
	}
}

func TestHubConcurrentBanOperations(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema for database operations
	CreateSchema(db)

	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	username := "testuser"
	adminUsername := "admin"

	// Test concurrent ban/unban operations
	done := make(chan bool, 2)

	// Goroutine 1: Ban and unban user
	go func() {
		for i := 0; i < 100; i++ {
			hub.BanUser(username, adminUsername)
			hub.UnbanUser(username, adminUsername)
		}
		done <- true
	}()

	// Goroutine 2: Check if user is banned
	go func() {
		for i := 0; i < 100; i++ {
			hub.IsUserBanned(username)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Final state should be consistent
	// The user should not be banned after the unban in the first goroutine
	if hub.IsUserBanned(username) {
		t.Error("User should not be banned after concurrent operations")
	}
}

func TestBroadcastUserListNonBlocking(t *testing.T) {
	hub := NewHub("", "", "", nil)

	// Create a client with a tiny send buffer that we intentionally fill.
	stalled := &Client{username: "stalled", send: make(chan interface{}, 1)}
	healthy := &Client{username: "healthy", send: make(chan interface{}, 10)}

	hub.clientsMutex.Lock()
	hub.clients[stalled] = true
	hub.clients[healthy] = true
	hub.clientsMutex.Unlock()

	// Fill the stalled client's buffer so the next send would block.
	stalled.send <- "filler"

	// broadcastUserList must not block even though stalled's buffer is full.
	done := make(chan struct{})
	go func() {
		hub.broadcastUserList()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("broadcastUserList blocked on full send channel")
	}

	if len(healthy.send) != 1 {
		t.Errorf("healthy client should have received user list, got %d messages", len(healthy.send))
	}
}

func TestKickUserNonBlocking(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()
	CreateSchema(db)

	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	// Create a client with a full send buffer.
	client := &Client{
		username: "victim",
		send:     make(chan interface{}, 1),
		conn:     nil, // conn.Close will be skipped via nil check in test
	}
	client.send <- "filler"

	hub.clientsMutex.Lock()
	hub.clients[client] = true
	hub.clientsMutex.Unlock()

	// kickUser must not block even though the buffer is full.
	done := make(chan struct{})
	go func() {
		defer func() {
			_ = recover() // conn is nil in test; tolerate nil pointer in conn.Close
			close(done)
		}()
		hub.kickUser("victim", "test")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("kickUser blocked on full send channel")
	}
}

func TestChannelManagement(t *testing.T) {
	hub := NewHub("", "", "", nil)

	client := &Client{username: "testuser", send: make(chan interface{}, 10)}

	t.Run("default channel is general", func(t *testing.T) {
		if got := hub.getClientChannel(client); got != "general" {
			t.Errorf("expected general, got %s", got)
		}
	})

	t.Run("joinChannel puts client in specified channel", func(t *testing.T) {
		hub.joinChannel(client, "room1")
		if got := hub.getClientChannel(client); got != "room1" {
			t.Errorf("expected room1, got %s", got)
		}
	})

	t.Run("leaveChannel removes client from a channel", func(t *testing.T) {
		hub.leaveChannel(client, "room1")
		if got := hub.getClientChannel(client); got != "general" {
			t.Errorf("expected general after leave, got %s", got)
		}
	})

	t.Run("leaveChannel cleans up empty channels", func(t *testing.T) {
		hub.joinChannel(client, "lonely")
		hub.leaveChannel(client, "lonely")
		hub.channelMutex.RLock()
		_, exists := hub.channels["lonely"]
		hub.channelMutex.RUnlock()
		if exists {
			t.Error("expected lonely channel to be removed when empty")
		}
		got := hub.listChannels()
		sort.Strings(got)
		want := []string{"general"}
		if len(got) != len(want) || got[0] != want[0] {
			t.Errorf("listChannels = %v, want %v", got, want)
		}
	})

	t.Run("listChannels returns all active channels", func(t *testing.T) {
		other := &Client{username: "other", send: make(chan interface{}, 10)}
		hub.joinChannel(client, "alpha")
		hub.joinChannel(other, "beta")
		got := append([]string(nil), hub.listChannels()...)
		sort.Strings(got)
		want := []string{"alpha", "beta"}
		if len(got) != len(want) {
			t.Fatalf("listChannels len = %d, want %d: %v", len(got), len(want), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("listChannels[%d] = %s, want %s", i, got[i], want[i])
			}
		}
		hub.leaveChannel(client, "alpha")
		hub.leaveChannel(other, "beta")
	})

	t.Run("getClientChannel returns correct channel after join", func(t *testing.T) {
		joiner := &Client{username: "joiner", send: make(chan interface{}, 10)}
		hub.joinChannel(joiner, "vip")
		if got := hub.getClientChannel(joiner); got != "vip" {
			t.Errorf("expected vip, got %s", got)
		}
		hub.leaveChannel(joiner, "vip")
	})

	t.Run("getChannelUsers returns correct users", func(t *testing.T) {
		a := &Client{username: "alice", send: make(chan interface{}, 10)}
		b := &Client{username: "bob", send: make(chan interface{}, 10)}
		hub.joinChannel(a, "dev")
		hub.joinChannel(b, "dev")
		users := hub.getChannelUsers("dev")
		if len(users) != 2 {
			t.Fatalf("expected 2 users, got %d", len(users))
		}
		hub.leaveChannel(a, "dev")
		users = hub.getChannelUsers("dev")
		if len(users) != 1 {
			t.Errorf("expected 1 user after leave, got %d", len(users))
		}
		hub.leaveChannel(b, "dev")
	})

	t.Run("getChannelUsers returns empty for nonexistent channel", func(t *testing.T) {
		users := hub.getChannelUsers("nonexistent")
		if len(users) != 0 {
			t.Errorf("expected 0 users, got %d", len(users))
		}
	})
}

func TestBroadcastDM(t *testing.T) {
	hub := NewHub("", "", "", nil)

	sender := &Client{username: "alice", send: make(chan interface{}, 10)}
	recipient := &Client{username: "bob", send: make(chan interface{}, 10)}
	bystander := &Client{username: "eve", send: make(chan interface{}, 10)}

	hub.clientsMutex.Lock()
	hub.clients[sender] = true
	hub.clients[recipient] = true
	hub.clients[bystander] = true
	hub.clientsMutex.Unlock()

	msg := shared.Message{
		Sender:    "alice",
		Recipient: "bob",
		Content:   "secret",
		Type:      shared.DirectMessage,
	}

	hub.broadcastDM(msg)

	if len(sender.send) != 1 {
		t.Errorf("sender should receive DM echo, got %d messages", len(sender.send))
	}
	if len(recipient.send) != 1 {
		t.Errorf("recipient should receive DM, got %d messages", len(recipient.send))
	}
	if len(bystander.send) != 0 {
		t.Errorf("bystander should not receive DM, got %d messages", len(bystander.send))
	}
}

func TestBroadcastDMCaseInsensitive(t *testing.T) {
	hub := NewHub("", "", "", nil)

	sender := &Client{username: "Alice", send: make(chan interface{}, 10)}
	recipient := &Client{username: "BOB", send: make(chan interface{}, 10)}

	hub.clientsMutex.Lock()
	hub.clients[sender] = true
	hub.clients[recipient] = true
	hub.clientsMutex.Unlock()

	msg := shared.Message{
		Sender:    "alice",
		Recipient: "bob",
		Content:   "hi",
		Type:      shared.DirectMessage,
	}

	hub.broadcastDM(msg)

	if len(sender.send) != 1 {
		t.Errorf("sender should receive DM (case insensitive), got %d", len(sender.send))
	}
	if len(recipient.send) != 1 {
		t.Errorf("recipient should receive DM (case insensitive), got %d", len(recipient.send))
	}
}

func TestConcurrentChannelOperations(t *testing.T) {
	hub := NewHub("", "", "", nil)

	done := make(chan bool, 4)
	clients := make([]*Client, 10)
	for i := range clients {
		clients[i] = &Client{username: "user" + string(rune('0'+i)), send: make(chan interface{}, 10)}
	}

	go func() {
		for i := 0; i < 50; i++ {
			hub.joinChannel(clients[i%len(clients)], "room")
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 50; i++ {
			hub.leaveChannel(clients[i%len(clients)], "room")
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 50; i++ {
			hub.listChannels()
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 50; i++ {
			hub.getChannelUsers("room")
		}
		done <- true
	}()

	for i := 0; i < 4; i++ {
		<-done
	}
}
