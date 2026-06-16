package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"
)

func TestIntegrationMessageFlow(t *testing.T) {
	// Create a test database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema
	CreateSchema(db)

	// Create hub (for future use in tests)
	_ = NewHub("./plugins", "./data", "http://registry.example.com", db)

	// Test message insertion and retrieval
	now := time.Now()
	testMessages := []shared.Message{
		{Sender: "alice", Content: "Hello Bob!", CreatedAt: now.Add(-2 * time.Hour), Encrypted: false},
		{Sender: "bob", Content: "Hi Alice!", CreatedAt: now.Add(-1 * time.Hour), Encrypted: false},
		{Sender: "alice", Content: "How are you?", CreatedAt: now, Encrypted: false},
	}

	for _, msg := range testMessages {
		if _, err := InsertMessage(db, msg); err != nil {
			t.Fatalf("InsertMessage failed: %v", err)
		}
	}

	// Retrieve messages
	recentMessages := GetRecentMessages(db)
	if len(recentMessages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(recentMessages))
	}

	// Verify message order (should be chronological)
	if recentMessages[0].Content != "Hello Bob!" {
		t.Errorf("Expected first message 'Hello Bob!', got '%s'", recentMessages[0].Content)
	}
	if recentMessages[1].Content != "Hi Alice!" {
		t.Errorf("Expected second message 'Hi Alice!', got '%s'", recentMessages[1].Content)
	}
	if recentMessages[2].Content != "How are you?" {
		t.Errorf("Expected third message 'How are you?', got '%s'", recentMessages[2].Content)
	}

	visible := GetRecentMessagesForUser(db, "bob", HandshakeReplayLimit, false)
	if len(visible) != 3 {
		t.Errorf("Expected 3 visible messages for bob, got %d", len(visible))
	}
}

func TestIntegrationUserBanFlow(t *testing.T) {
	// Create a test database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema
	CreateSchema(db)

	// Create hub
	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	username := "troublemaker"
	adminUsername := "admin"

	// User should not be banned initially
	if hub.IsUserBanned(username) {
		t.Error("User should not be banned initially")
	}

	// Ban the user
	hub.BanUser(username, adminUsername)
	if !hub.IsUserBanned(username) {
		t.Error("User should be banned after BanUser")
	}

	// Test case insensitive ban check
	if !hub.IsUserBanned(strings.ToUpper(username)) {
		t.Error("Ban should be case insensitive")
	}

	// Unban the user
	unbanned := hub.UnbanUser(username, adminUsername)
	if !unbanned {
		t.Error("UnbanUser should return true")
	}

	if hub.IsUserBanned(username) {
		t.Error("User should not be banned after UnbanUser")
	}

	// Test kick flow
	hub.KickUser(username, adminUsername)
	if !hub.IsUserBanned(username) {
		t.Error("User should be kicked after KickUser")
	}

	// Allow user back
	allowed := hub.AllowUser(username, adminUsername)
	if !allowed {
		t.Error("AllowUser should return true")
	}

	if hub.IsUserBanned(username) {
		t.Error("User should not be banned after AllowUser")
	}
}

func TestIntegrationDatabaseStats(t *testing.T) {
	// Create a test database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema
	CreateSchema(db)

	// Insert various types of messages
	now := time.Now()
	messages := []shared.Message{
		{Sender: "user1", Content: "Message 1", CreatedAt: now.Add(-3 * time.Hour), Encrypted: false},
		{Sender: "user2", Content: "Message 2", CreatedAt: now.Add(-2 * time.Hour), Encrypted: false},
		{Sender: "user1", Content: "Message 3", CreatedAt: now.Add(-1 * time.Hour), Encrypted: false},
		{Sender: "System", Content: "System message", CreatedAt: now, Encrypted: false},
	}

	for _, msg := range messages {
		if _, err := InsertMessage(db, msg); err != nil {
			t.Fatalf("InsertMessage failed: %v", err)
		}
	}

	// Get database stats
	stats, err := GetDatabaseStats(db)
	if err != nil {
		t.Fatalf("GetDatabaseStats failed: %v", err)
	}

	// Verify stats content
	if !strings.Contains(stats, "Total Messages: 4") {
		t.Errorf("Expected 'Total Messages: 4' in stats, got: %s", stats)
	}

	if !strings.Contains(stats, "Unique Users: 2") { // user1, user2 (System excluded from user count)
		t.Errorf("Expected 'Unique Users: 2' in stats, got: %s", stats)
	}

	if !strings.Contains(stats, "Database Statistics:") {
		t.Errorf("Expected 'Database Statistics:' in stats, got: %s", stats)
	}
}

func TestIntegrationEncryptedMessageFlow(t *testing.T) {
	// Create a test database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema
	CreateSchema(db)

	// Create an encrypted message
	encryptedMsg := &shared.EncryptedMessage{
		Sender:      "alice",
		Content:     "Secret message",
		CreatedAt:   time.Now(),
		IsEncrypted: true,
		Encrypted:   []byte("encrypted data here"),
		Nonce:       []byte("nonce data"),
		Recipient:   "bob",
	}

	// Insert encrypted message
	if err := InsertEncryptedMessage(db, encryptedMsg); err != nil {
		t.Fatalf("InsertEncryptedMessage failed: %v", err)
	}

	// Retrieve messages (this would need to be modified to handle encrypted messages properly)
	recentMessages := GetRecentMessages(db)
	if len(recentMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(recentMessages))
	}

	if recentMessages[0].Sender != "alice" {
		t.Errorf("Expected sender 'alice', got '%s'", recentMessages[0].Sender)
	}

	if !recentMessages[0].Encrypted {
		t.Error("Message should be marked as encrypted")
	}
}

func TestIntegrationMessageCap(t *testing.T) {
	// Create a test database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema
	CreateSchema(db)

	// Insert more than 1000 messages (the cap limit)
	for i := 0; i < 1100; i++ {
		msg := shared.Message{
			Sender:    "user",
			Content:   fmt.Sprintf("Message %d", i),
			CreatedAt: time.Now().Add(time.Duration(i) * time.Minute),
			Encrypted: false,
		}
		if _, err := InsertMessage(db, msg); err != nil {
			t.Fatalf("InsertMessage failed: %v", err)
		}
	}

	// Retrieve recent messages
	recentMessages := GetRecentMessages(db)

	// Should only have 50 recent messages (limit in GetRecentMessages)
	if len(recentMessages) != 50 {
		t.Errorf("Expected 50 recent messages, got %d", len(recentMessages))
	}

	// Verify we have the most recent messages
	// The messages should be sorted chronologically, so the last message
	// should be the one with the highest number
	if !strings.Contains(recentMessages[len(recentMessages)-1].Content, "Message 1099") {
		t.Errorf("Expected most recent message to be 'Message 1099', got '%s'",
			recentMessages[len(recentMessages)-1].Content)
	}
}

func TestIntegrationWebSocketHandshakeReplayOnReconnect(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()
	CreateSchema(db)

	now := time.Now()
	for i, content := range []string{"replay-one", "replay-two", "replay-three"} {
		if _, err := InsertMessage(db, shared.Message{
			Sender:    "alice",
			Content:   content,
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("InsertMessage: %v", err)
		}
	}

	tdir := t.TempDir()
	hub := NewHub(tdir, tdir, "", db)
	go hub.Run()

	handler := ServeWs(hub, db, nil, "admin-key", false, 10<<20, filepath.Join(tdir, "test.db"))
	srv := httptest.NewServer(handler)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	readHandshakeHistory := func(username string) map[string]bool {
		t.Helper()
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("dial websocket: %v", err)
		}
		defer conn.Close()

		if err := conn.WriteJSON(shared.Handshake{Username: username}); err != nil {
			t.Fatalf("write handshake: %v", err)
		}

		got := make(map[string]bool)
		wantCount := 3
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			_, raw, err := conn.ReadMessage()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					if len(got) >= wantCount {
						break
					}
					continue
				}
				break
			}

			var chat shared.Message
			if err := json.Unmarshal(raw, &chat); err == nil && chat.Sender != "" && chat.Sender != "System" && chat.Content != "" {
				got[chat.Content] = true
			}
		}
		time.Sleep(100 * time.Millisecond)
		return got
	}

	first := readHandshakeHistory("viewer")
	for _, want := range []string{"replay-one", "replay-two", "replay-three"} {
		if !first[want] {
			t.Fatalf("first connect missing %q; got %#v", want, first)
		}
	}

	second := readHandshakeHistory("viewer")
	for _, want := range []string{"replay-one", "replay-two", "replay-three"} {
		if !second[want] {
			t.Fatalf("reconnect missing %q; first=%#v second=%#v", want, first, second)
		}
	}
}

func TestIntegrationConcurrentOperations(t *testing.T) {
	// Create a test database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema
	CreateSchema(db)

	// Create hub
	hub := NewHub("./plugins", "./data", "http://registry.example.com", db)

	// Test concurrent message insertions with proper synchronization
	var wg sync.WaitGroup
	var dbMutex sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := shared.Message{
				Sender:    fmt.Sprintf("user%d", id),
				Content:   fmt.Sprintf("Message from user %d", id),
				CreatedAt: time.Now(),
				Encrypted: false,
			}
			// Synchronize database access
			dbMutex.Lock()
			if _, err := InsertMessage(db, msg); err != nil {
				t.Errorf("InsertMessage failed: %v", err)
			}
			dbMutex.Unlock()
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify all messages were inserted
	recentMessages := GetRecentMessages(db)
	if len(recentMessages) != 10 {
		t.Errorf("Expected 10 messages, got %d", len(recentMessages))
	}

	// Test concurrent ban operations with proper synchronization
	var banWg sync.WaitGroup

	for i := 0; i < 5; i++ {
		banWg.Add(1)
		go func(id int) {
			defer banWg.Done()
			username := fmt.Sprintf("user%d", id)
			hub.BanUser(username, "admin")
			hub.UnbanUser(username, "admin")
		}(i)
	}

	// Wait for all ban operations to complete
	banWg.Wait()

	// Verify no users are banned after unban operations
	for i := 0; i < 5; i++ {
		username := fmt.Sprintf("user%d", i)
		if hub.IsUserBanned(username) {
			t.Errorf("User %s should not be banned after unban", username)
		}
	}
}
