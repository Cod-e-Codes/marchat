package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
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

	// Insert messages
	for _, msg := range testMessages {
		if err := InsertMessage(db, msg); err != nil {
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

	// Test GetMessagesAfter
	messagesAfter := GetMessagesAfter(db, 1, 10)
	if len(messagesAfter) != 2 {
		t.Errorf("Expected 2 messages after ID 1, got %d", len(messagesAfter))
	}

	if messagesAfter[0].Content != "Hi Alice!" {
		t.Errorf("Expected first message after ID 1 'Hi Alice!', got '%s'", messagesAfter[0].Content)
	}
	if messagesAfter[1].Content != "How are you?" {
		t.Errorf("Expected second message after ID 1 'How are you?', got '%s'", messagesAfter[1].Content)
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
		if err := InsertMessage(db, msg); err != nil {
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
		if err := InsertMessage(db, msg); err != nil {
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

func TestIntegrationWebSocketHandshake(t *testing.T) {
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

	// Test admin authentication
	adminList := []string{"admin1", "admin2"}
	adminKey := "secret-admin-key"
	banGapsHistory := false
	maxFileBytes := int64(10 * 1024 * 1024) // 10MB

	// Create handler
	dbPath := "test_marchat.db"
	handler := ServeWs(hub, db, adminList, adminKey, banGapsHistory, maxFileBytes, dbPath)

	// Test regular user handshake
	handshake := shared.Handshake{
		Username: "regularuser",
		Admin:    false,
		AdminKey: "",
	}

	handshakeData, err := json.Marshal(handshake)
	if err != nil {
		t.Fatalf("Failed to marshal handshake: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/ws", strings.NewReader(string(handshakeData)))
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "test-key")
	req.Header.Set("Sec-WebSocket-Version", "13")

	// Create response recorder
	w := httptest.NewRecorder()

	// Note: This test is simplified - actual WebSocket testing would require
	// a WebSocket client and more complex setup
	handler.ServeHTTP(w, req)

	// The handler should attempt to upgrade the connection
	// In a real test, we'd verify the WebSocket upgrade response
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
			if err := InsertMessage(db, msg); err != nil {
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
