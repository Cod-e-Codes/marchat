package server

import (
	"database/sql"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
	_ "modernc.org/sqlite"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name           string
		request        *http.Request
		expectedIP     string
		expectedResult string
	}{
		{
			name: "X-Forwarded-For single IP",
			request: &http.Request{
				Header: http.Header{
					"X-Forwarded-For": []string{"192.168.1.1"},
				},
			},
			expectedResult: "192.168.1.1",
		},
		{
			name: "X-Forwarded-For multiple IPs",
			request: &http.Request{
				Header: http.Header{
					"X-Forwarded-For": []string{"192.168.1.1, 10.0.0.1, 172.16.0.1"},
				},
			},
			expectedResult: "192.168.1.1",
		},
		{
			name: "X-Real-IP header",
			request: &http.Request{
				Header: http.Header{
					"X-Real-Ip": []string{"203.0.113.1"},
				},
			},
			expectedResult: "203.0.113.1",
		},
		{
			name: "RemoteAddr fallback",
			request: &http.Request{
				RemoteAddr: "192.168.1.100:12345",
			},
			expectedResult: "192.168.1.100",
		},
		{
			name: "RemoteAddr without port",
			request: &http.Request{
				RemoteAddr: "192.168.1.100",
			},
			expectedResult: "192.168.1.100",
		},
		{
			name:           "No IP information",
			request:        &http.Request{},
			expectedResult: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getClientIP(tt.request)
			if result != tt.expectedResult {
				t.Errorf("Expected IP %s, got %s", tt.expectedResult, result)
			}
		})
	}
}

func TestInsertMessage(t *testing.T) {
	// Create a real in-memory database for testing
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema
	CreateSchema(db)

	msg := shared.Message{
		Sender:    "testuser",
		Content:   "Hello, World!",
		CreatedAt: time.Now(),
		Encrypted: false,
	}

	// Insert message
	if err := InsertMessage(db, msg); err != nil {
		t.Fatalf("InsertMessage failed: %v", err)
	}

	// Verify message was inserted
	recentMessages := GetRecentMessages(db)
	if len(recentMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(recentMessages))
	}

	if recentMessages[0].Sender != msg.Sender {
		t.Errorf("Expected sender %s, got %s", msg.Sender, recentMessages[0].Sender)
	}

	if recentMessages[0].Content != msg.Content {
		t.Errorf("Expected content %s, got %s", msg.Content, recentMessages[0].Content)
	}

	if recentMessages[0].Encrypted != msg.Encrypted {
		t.Errorf("Expected encrypted %v, got %v", msg.Encrypted, recentMessages[0].Encrypted)
	}
}

func TestInsertEncryptedMessage(t *testing.T) {
	// Create a real in-memory database for testing
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema
	CreateSchema(db)

	encryptedMsg := &shared.EncryptedMessage{
		Sender:      "testuser",
		Content:     "encrypted content",
		CreatedAt:   time.Now(),
		IsEncrypted: true,
		Encrypted:   []byte("encrypted data"),
		Nonce:       []byte("nonce"),
		Recipient:   "recipient",
	}

	// Insert encrypted message
	if err := InsertEncryptedMessage(db, encryptedMsg); err != nil {
		t.Fatalf("InsertEncryptedMessage failed: %v", err)
	}

	// Verify message was inserted
	recentMessages := GetRecentMessages(db)
	if len(recentMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(recentMessages))
	}

	if recentMessages[0].Sender != encryptedMsg.Sender {
		t.Errorf("Expected sender %s, got %s", encryptedMsg.Sender, recentMessages[0].Sender)
	}

	if !recentMessages[0].Encrypted {
		t.Error("Expected message to be marked as encrypted")
	}
}

func TestGetRecentMessages(t *testing.T) {
	// Create a real database for this test
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema
	CreateSchema(db)

	// Insert test messages
	now := time.Now()
	messages := []shared.Message{
		{Sender: "user1", Content: "First message", CreatedAt: now.Add(-2 * time.Hour), Encrypted: false},
		{Sender: "user2", Content: "Second message", CreatedAt: now.Add(-1 * time.Hour), Encrypted: false},
		{Sender: "user1", Content: "Third message", CreatedAt: now, Encrypted: false},
	}

	for _, msg := range messages {
		if err := InsertMessage(db, msg); err != nil {
			t.Fatalf("InsertMessage failed: %v", err)
		}
	}

	// Get recent messages
	recentMessages := GetRecentMessages(db)

	if len(recentMessages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(recentMessages))
	}

	// Messages should be sorted chronologically (oldest first)
	if recentMessages[0].Content != "First message" {
		t.Errorf("Expected first message 'First message', got '%s'", recentMessages[0].Content)
	}

	if recentMessages[1].Content != "Second message" {
		t.Errorf("Expected second message 'Second message', got '%s'", recentMessages[1].Content)
	}

	if recentMessages[2].Content != "Third message" {
		t.Errorf("Expected third message 'Third message', got '%s'", recentMessages[2].Content)
	}
}

func TestGetMessagesAfter(t *testing.T) {
	// Create a real database for this test
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema
	CreateSchema(db)

	// Insert test messages
	now := time.Now()
	messages := []shared.Message{
		{Sender: "user1", Content: "Message 1", CreatedAt: now.Add(-3 * time.Hour), Encrypted: false},
		{Sender: "user2", Content: "Message 2", CreatedAt: now.Add(-2 * time.Hour), Encrypted: false},
		{Sender: "user1", Content: "Message 3", CreatedAt: now.Add(-1 * time.Hour), Encrypted: false},
		{Sender: "user2", Content: "Message 4", CreatedAt: now, Encrypted: false},
	}

	for _, msg := range messages {
		if err := InsertMessage(db, msg); err != nil {
			t.Fatalf("InsertMessage failed: %v", err)
		}
	}

	// Get messages after the first one (message_id = 1)
	messagesAfter := GetMessagesAfter(db, 1, 10)

	if len(messagesAfter) != 3 {
		t.Errorf("Expected 3 messages after ID 1, got %d", len(messagesAfter))
	}

	// Messages should be sorted chronologically
	if messagesAfter[0].Content != "Message 2" {
		t.Errorf("Expected first message 'Message 2', got '%s'", messagesAfter[0].Content)
	}

	if messagesAfter[1].Content != "Message 3" {
		t.Errorf("Expected second message 'Message 3', got '%s'", messagesAfter[1].Content)
	}

	if messagesAfter[2].Content != "Message 4" {
		t.Errorf("Expected third message 'Message 4', got '%s'", messagesAfter[2].Content)
	}
}

func TestSortMessagesByTimestamp(t *testing.T) {
	now := time.Now()
	messages := []shared.Message{
		{Sender: "user1", Content: "Third", CreatedAt: now, Encrypted: false},
		{Sender: "user2", Content: "First", CreatedAt: now.Add(-2 * time.Hour), Encrypted: false},
		{Sender: "user1", Content: "Second", CreatedAt: now.Add(-1 * time.Hour), Encrypted: false},
	}

	// Sort the messages
	sortMessagesByTimestamp(messages)

	// Check order
	if messages[0].Content != "First" {
		t.Errorf("Expected first message 'First', got '%s'", messages[0].Content)
	}

	if messages[1].Content != "Second" {
		t.Errorf("Expected second message 'Second', got '%s'", messages[1].Content)
	}

	if messages[2].Content != "Third" {
		t.Errorf("Expected third message 'Third', got '%s'", messages[2].Content)
	}
}

func TestSortMessagesByTimestampWithSameTime(t *testing.T) {
	now := time.Now()
	messages := []shared.Message{
		{Sender: "user2", Content: "Second", CreatedAt: now, Encrypted: false},
		{Sender: "user1", Content: "First", CreatedAt: now, Encrypted: false},
	}

	// Sort the messages
	sortMessagesByTimestamp(messages)

	// With same timestamp, should sort by sender alphabetically
	if messages[0].Sender != "user1" {
		t.Errorf("Expected first message sender 'user1', got '%s'", messages[0].Sender)
	}

	if messages[1].Sender != "user2" {
		t.Errorf("Expected second message sender 'user2', got '%s'", messages[1].Sender)
	}
}

func TestClearMessages(t *testing.T) {
	// Create a real database for this test
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema
	CreateSchema(db)

	// Insert test message
	msg := shared.Message{
		Sender:    "testuser",
		Content:   "Test message",
		CreatedAt: time.Now(),
		Encrypted: false,
	}
	if err := InsertMessage(db, msg); err != nil {
		t.Fatalf("InsertMessage failed: %v", err)
	}

	// Verify message exists
	recentMessages := GetRecentMessages(db)
	if len(recentMessages) != 1 {
		t.Errorf("Expected 1 message before clear, got %d", len(recentMessages))
	}

	// Clear messages
	err = ClearMessages(db)
	if err != nil {
		t.Fatalf("ClearMessages failed: %v", err)
	}

	// Verify messages are cleared
	recentMessages = GetRecentMessages(db)
	if len(recentMessages) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(recentMessages))
	}
}

func TestBackupDatabase(t *testing.T) {
	// Create a temporary file for the test database
	tempDB := t.TempDir() + "/test.db"

	// Create a test database
	db, err := sql.Open("sqlite", tempDB)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	CreateSchema(db)

	// Insert test data
	msg := shared.Message{
		Sender:    "testuser",
		Content:   "Backup test message",
		CreatedAt: time.Now(),
		Encrypted: false,
	}
	if err := InsertMessage(db, msg); err != nil {
		t.Fatalf("InsertMessage failed: %v", err)
	}

	// Close the database before backup
	db.Close()

	// Test backup
	backupFilename, err := BackupDatabase(tempDB)
	if err != nil {
		t.Fatalf("BackupDatabase failed: %v", err)
	}

	if backupFilename == "" {
		t.Error("Expected backup filename, got empty string")
	}

	// Verify backup file exists
	if !strings.Contains(backupFilename, "marchat_backup_") {
		t.Errorf("Expected backup filename to contain 'marchat_backup_', got '%s'", backupFilename)
	}

	if !strings.HasSuffix(backupFilename, ".db") {
		t.Errorf("Expected backup filename to end with '.db', got '%s'", backupFilename)
	}
}

func TestGetDatabaseStats(t *testing.T) {
	// Create a real database for this test
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create schema
	CreateSchema(db)

	// Insert test messages
	now := time.Now()
	messages := []shared.Message{
		{Sender: "user1", Content: "Message 1", CreatedAt: now.Add(-2 * time.Hour), Encrypted: false},
		{Sender: "user2", Content: "Message 2", CreatedAt: now.Add(-1 * time.Hour), Encrypted: false},
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

	if !strings.Contains(stats, "Total Messages: 3") {
		t.Errorf("Expected stats to contain 'Total Messages: 3', got: %s", stats)
	}

	if !strings.Contains(stats, "Unique Users: 2") {
		t.Errorf("Expected stats to contain 'Unique Users: 2', got: %s", stats)
	}

	if !strings.Contains(stats, "Database Statistics:") {
		t.Errorf("Expected stats to contain 'Database Statistics:', got: %s", stats)
	}
}
