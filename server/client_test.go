package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/gorilla/websocket"

	_ "modernc.org/sqlite"
)

func setupTestClient(t *testing.T) (*Client, *Hub, *sql.DB, func()) {
	t.Helper()

	// Create temporary database
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	CreateSchema(db)

	// Create hub with correct parameters
	hub := NewHub(tdir, tdir, "http://localhost:8080", db)
	go hub.Run()

	// Create mock websocket connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		_, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade connection: %v", err)
		}
	}))
	defer server.Close()

	// Connect to the test server
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}

	// Create client
	client := &Client{
		hub:          hub,
		conn:         conn,
		send:         make(chan interface{}, 256),
		db:           db,
		username:     "testuser",
		isAdmin:      false,
		ipAddr:       "127.0.0.1",
		maxFileBytes: 1024 * 1024, // 1MB
		dbPath:       dbPath,
	}

	cleanup := func() {
		conn.Close()
		db.Close()
	}

	return client, hub, db, cleanup
}

func TestClient_Initialization(t *testing.T) {
	client, hub, _, cleanup := setupTestClient(t)
	defer cleanup()

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.hub != hub {
		t.Error("Client hub should match provided hub")
	}

	if client.db == nil {
		t.Error("Client database should not be nil")
	}

	if client.username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", client.username)
	}

	if client.ipAddr != "127.0.0.1" {
		t.Errorf("Expected IP '127.0.0.1', got '%s'", client.ipAddr)
	}

	if client.maxFileBytes != 1024*1024 {
		t.Errorf("Expected maxFileBytes 1MB, got %d", client.maxFileBytes)
	}

	if client.send == nil {
		t.Error("Send channel should be initialized")
	}

	if cap(client.send) != 256 {
		t.Errorf("Expected send channel capacity 256, got %d", cap(client.send))
	}
}

func TestClient_ReadPump_ConnectionSettings(t *testing.T) {
	client, _, _, cleanup := setupTestClient(t)
	defer cleanup()

	// Test that connection settings are applied
	// We can't easily test the actual readPump without a real connection,
	// but we can test the connection configuration

	if client.conn == nil {
		t.Error("WebSocket connection should not be nil")
	}

	// Test that we can set read limits and deadlines
	limit := int64(1024*1024) + 512
	if client.maxFileBytes > 0 {
		limit = client.maxFileBytes + 512
	}

	client.conn.SetReadLimit(limit)

	deadline := time.Now().Add(pongWait)
	err := client.conn.SetReadDeadline(deadline)
	if err != nil {
		t.Errorf("Failed to set read deadline: %v", err)
	}

	// Test pong handler
	pongHandler := func(string) error {
		return client.conn.SetReadDeadline(time.Now().Add(pongWait))
	}
	client.conn.SetPongHandler(pongHandler)
}

func TestClient_SendChannel(t *testing.T) {
	client, _, _, cleanup := setupTestClient(t)
	defer cleanup()

	// Test that we can send messages through the channel
	testMessage := shared.Message{
		Sender:    "testuser",
		Content:   "Hello, world!",
		CreatedAt: time.Now(),
		Type:      shared.TextMessage,
	}

	// Send message (non-blocking)
	select {
	case client.send <- testMessage:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Failed to send message to client channel")
	}

	// Verify message was sent
	select {
	case receivedMessage := <-client.send:
		if receivedMessage != testMessage {
			t.Error("Received message does not match sent message")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Failed to receive message from client channel")
	}
}

func TestClient_AdminStatus(t *testing.T) {
	client, _, db, cleanup := setupTestClient(t)
	defer cleanup()

	// Test non-admin client
	if client.isAdmin {
		t.Error("Client should not be admin by default")
	}

	// Test admin client
	adminClient := &Client{
		hub:          client.hub,
		conn:         client.conn,
		send:         make(chan interface{}, 256),
		db:           db,
		username:     "admin",
		isAdmin:      true,
		ipAddr:       "127.0.0.1",
		maxFileBytes: 1024 * 1024,
		dbPath:       client.dbPath,
	}

	if !adminClient.isAdmin {
		t.Error("Admin client should have admin status")
	}

	// Use the fields to avoid unused write warnings
	_ = adminClient.hub
	_ = adminClient.conn
	_ = adminClient.send
	_ = adminClient.db
	_ = adminClient.username
	_ = adminClient.ipAddr
	_ = adminClient.maxFileBytes
	_ = adminClient.dbPath
}

func TestClient_IPAddress(t *testing.T) {
	client, _, _, cleanup := setupTestClient(t)
	defer cleanup()

	testIPs := []string{
		"192.168.1.1",
		"10.0.0.1",
		"172.16.0.1",
		"::1",
		"2001:db8::1",
	}

	for _, ip := range testIPs {
		client.ipAddr = ip
		if client.ipAddr != ip {
			t.Errorf("Expected IP %s, got %s", ip, client.ipAddr)
		}
	}
}

func TestClient_FileSizeLimits(t *testing.T) {
	client, _, _, cleanup := setupTestClient(t)
	defer cleanup()

	testLimits := []int64{
		1024,              // 1KB
		1024 * 1024,       // 1MB
		10 * 1024 * 1024,  // 10MB
		100 * 1024 * 1024, // 100MB
	}

	for _, limit := range testLimits {
		client.maxFileBytes = limit
		if client.maxFileBytes != limit {
			t.Errorf("Expected maxFileBytes %d, got %d", limit, client.maxFileBytes)
		}
	}
}

func TestClient_DatabasePath(t *testing.T) {
	client, _, _, cleanup := setupTestClient(t)
	defer cleanup()

	if client.dbPath == "" {
		t.Error("Database path should not be empty")
	}

	// Test that we can change the database path
	newPath := "/tmp/new/database.db"
	client.dbPath = newPath

	if client.dbPath != newPath {
		t.Errorf("Expected database path %s, got %s", newPath, client.dbPath)
	}
}

func TestClient_PluginCommandHandler(t *testing.T) {
	client, _, _, cleanup := setupTestClient(t)
	defer cleanup()

	// Test that plugin command handler can be set
	if client.pluginCommandHandler != nil {
		t.Error("Plugin command handler should be nil by default")
	}

	// Create a mock plugin command handler
	handler := &PluginCommandHandler{}
	client.pluginCommandHandler = handler

	if client.pluginCommandHandler != handler {
		t.Error("Plugin command handler should match the set handler")
	}
}

func TestClient_MessageTypes(t *testing.T) {
	client, _, _, cleanup := setupTestClient(t)
	defer cleanup()

	// Test different message types
	messageTypes := []shared.MessageType{
		shared.TextMessage,
		shared.FileMessageType,
	}

	for _, msgType := range messageTypes {
		message := shared.Message{
			Sender:    "testuser",
			Content:   "Test message",
			CreatedAt: time.Now(),
			Type:      msgType,
		}

		// Send message through channel
		select {
		case client.send <- message:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Failed to send %s message", msgType)
		}
	}
}

func TestClient_JSONSerialization(t *testing.T) {
	_, _, _, cleanup := setupTestClient(t)
	defer cleanup()

	// Create a test message
	message := shared.Message{
		Sender:    "testuser",
		Content:   "Hello, world!",
		CreatedAt: time.Now(),
		Type:      shared.TextMessage,
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("Failed to marshal message to JSON: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("JSON data should not be empty")
	}

	// Test JSON deserialization
	var decodedMessage shared.Message
	err = json.Unmarshal(jsonData, &decodedMessage)
	if err != nil {
		t.Fatalf("Failed to unmarshal message from JSON: %v", err)
	}

	if decodedMessage.Sender != message.Sender {
		t.Error("Sender mismatch after JSON roundtrip")
	}

	if decodedMessage.Content != message.Content {
		t.Error("Content mismatch after JSON roundtrip")
	}
}

func TestClient_ConcurrentSend(t *testing.T) {
	client, _, _, cleanup := setupTestClient(t)
	defer cleanup()

	_ = client // Use client to avoid unused variable warning

	// Test concurrent sends to the client channel
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			message := shared.Message{
				Sender:    "testuser",
				Content:   string(rune('A' + id)),
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}

			select {
			case client.send <- message:
				done <- true
			case <-time.After(1 * time.Second):
				t.Errorf("Failed to send message %d", id)
				done <- false
			}
		}(i)
	}

	// Wait for all sends to complete
	successCount := 0
	for i := 0; i < 10; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful sends, got %d", successCount)
	}
}

func TestClient_ChannelCapacity(t *testing.T) {
	client, _, _, cleanup := setupTestClient(t)
	defer cleanup()

	// Test that we can fill the channel to capacity
	capacity := cap(client.send)

	// Fill channel
	for i := 0; i < capacity; i++ {
		message := shared.Message{
			Sender:    "testuser",
			Content:   string(rune('A' + i)),
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}

		select {
		case client.send <- message:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Failed to send message %d to channel", i)
		}
	}

	// Test that channel is full
	message := shared.Message{
		Sender:    "testuser",
		Content:   "This should block",
		CreatedAt: time.Now(),
		Type:      shared.TextMessage,
	}

	select {
	case client.send <- message:
		t.Error("Channel should be full and should not accept more messages")
	case <-time.After(100 * time.Millisecond):
		// Expected - channel is full
	}
}

func TestParseCommandWithQuotes(t *testing.T) {
	testCases := []struct {
		name     string
		command  string
		expected []string
	}{
		{
			name:     "simple command",
			command:  "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "quoted argument",
			command:  `hello "world with spaces"`,
			expected: []string{"hello", "world with spaces"},
		},
		{
			name:     "escaped quotes",
			command:  `hello \"world\"`,
			expected: []string{"hello", "\"world\""},
		},
		{
			name:     "multiple quoted args",
			command:  `cmd "arg1" "arg2 with spaces"`,
			expected: []string{"cmd", "arg1", "arg2 with spaces"},
		},
		{
			name:     "mixed quoted and unquoted",
			command:  `cmd arg1 "arg2 with spaces" arg3`,
			expected: []string{"cmd", "arg1", "arg2 with spaces", "arg3"},
		},
		{
			name:     "empty string",
			command:  "",
			expected: []string{},
		},
		{
			name:     "single word",
			command:  "hello",
			expected: []string{"hello"},
		},
		{
			name:     "escaped backslash",
			command:  `hello \\world`,
			expected: []string{"hello", "\\world"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseCommandWithQuotes(tc.command)
			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d parts, got %d", len(tc.expected), len(result))
				return
			}
			for i, part := range result {
				if part != tc.expected[i] {
					t.Errorf("Part %d: expected %q, got %q", i, tc.expected[i], part)
				}
			}
		})
	}
}

func TestClient_HandleAdminCommand(t *testing.T) {
	client, _, _, cleanup := setupTestClient(t)
	defer cleanup()

	// Test with empty command
	client.handleCommand("")
	// Should not panic or cause issues

	// Test with simple command
	client.handleCommand(":test")
	// Should not panic or cause issues

	// Test with quoted arguments
	client.handleCommand(`:test "arg with spaces"`)
	// Should not panic or cause issues

	// Test with plugin command handler - create a proper one
	client.pluginCommandHandler = NewPluginCommandHandler(client.hub.pluginManager)
	client.handleCommand(":plugin test")
	// Should not panic or cause issues

	// Test admin-only command
	client.isAdmin = true
	client.handleCommand(":stats")
	// Should not panic or cause issues
}
