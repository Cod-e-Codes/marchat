package server

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/gorilla/websocket"
)

// readSystemReplyContaining reads until a System message containing substr arrives.
// Any ReadJSON error ends the search (gorilla treats read errors, including
// deadlines, as permanent - do not retry after err).
func readSystemReplyContaining(t *testing.T, conn *websocket.Conn, substr string, timeout time.Duration) bool {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	for {
		var msg shared.Message
		if err := conn.ReadJSON(&msg); err != nil {
			return false
		}
		if msg.Sender == "System" && strings.Contains(msg.Content, substr) {
			return true
		}
	}
}

func TestIntegrationEmptyTextRejectedNoBroadcast(t *testing.T) {
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer db.Close()
	CreateSchema(db)

	hub := NewHub(tdir, tdir, "", db)
	go hub.Run()

	handler := ServeWs(hub, db, nil, "admin-key", false, 10<<20, dbPath)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	dial := func(username string) *websocket.Conn {
		t.Helper()
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		if err := conn.WriteJSON(shared.Handshake{Username: username}); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		return conn
	}

	listener := dial("emptyListen1")
	defer listener.Close()
	drainWSJSONUntilIdle(listener, 200*time.Millisecond)

	sender := dial("emptySend1")
	defer sender.Close()
	// Do not drain sender: deadline errors are permanent on gorilla Conn, and we
	// still need to read the System reject reply below.
	time.Sleep(100 * time.Millisecond)
	drainWSJSONUntilIdle(listener, 200*time.Millisecond)

	bad := shared.Message{
		Content: "   ",
		Type:    shared.TextMessage,
	}
	if err := sender.WriteJSON(bad); err != nil {
		t.Fatalf("send: %v", err)
	}

	if !readSystemReplyContaining(t, sender, "empty content", time.Second) {
		t.Fatal("expected System reply about empty content")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM messages WHERE TRIM(content) = '' OR content IS NULL`).Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 0 {
		t.Fatalf("empty content must not be persisted, got %d rows", count)
	}

	_ = listener.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
	var peerMsg shared.Message
	if err := listener.ReadJSON(&peerMsg); err == nil && peerMsg.Sender == "emptySend1" {
		t.Fatal("peer should not receive sender's empty message")
	}
}

func TestIntegrationEmptyDMRejected(t *testing.T) {
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer db.Close()
	CreateSchema(db)

	hub := NewHub(tdir, tdir, "", db)
	go hub.Run()

	handler := ServeWs(hub, db, nil, "admin-key", false, 10<<20, dbPath)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	dial := func(username string) *websocket.Conn {
		t.Helper()
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		if err := conn.WriteJSON(shared.Handshake{Username: username}); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		return conn
	}

	peer := dial("emptyDMPeer")
	defer peer.Close()
	drainWSJSONUntilIdle(peer, 200*time.Millisecond)

	sender := dial("emptyDMSend")
	defer sender.Close()
	time.Sleep(100 * time.Millisecond)
	drainWSJSONUntilIdle(peer, 200*time.Millisecond)

	bad := shared.Message{
		Content:   "",
		Type:      shared.DirectMessage,
		Recipient: "emptyDMPeer",
	}
	if err := sender.WriteJSON(bad); err != nil {
		t.Fatalf("send: %v", err)
	}

	if !readSystemReplyContaining(t, sender, "empty content", time.Second) {
		t.Fatal("expected System reply about empty content")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM messages WHERE recipient = ?`, "emptyDMPeer").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 0 {
		t.Fatalf("empty DM must not be persisted, got %d rows", count)
	}

	_ = peer.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
	var peerMsg shared.Message
	if err := peer.ReadJSON(&peerMsg); err == nil && peerMsg.Sender == "emptyDMSend" {
		t.Fatal("peer should not receive rejected empty DM")
	}
}

func TestIntegrationEmptyEditRejected(t *testing.T) {
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer db.Close()
	CreateSchema(db)

	hub := NewHub(tdir, tdir, "", db)
	go hub.Run()

	handler := ServeWs(hub, db, nil, "admin-key", false, 10<<20, dbPath)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(shared.Handshake{Username: "emptyEdit1"}); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	orig := shared.Message{
		Content: "keep-me",
		Type:    shared.TextMessage,
	}
	if err := conn.WriteJSON(orig); err != nil {
		t.Fatalf("send orig: %v", err)
	}
	// Wait for persist without draining (deadline would poison this Conn).
	deadline := time.Now().Add(time.Second)
	var msgID int64
	for time.Now().Before(deadline) {
		err := db.QueryRow(`SELECT message_id FROM messages WHERE content = ?`, "keep-me").Scan(&msgID)
		if err == nil && msgID > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if msgID <= 0 {
		t.Fatal("lookup message_id: original message not persisted")
	}

	edit := shared.Message{
		Content:   "  ",
		Type:      shared.EditMessageType,
		MessageID: msgID,
	}
	if err := conn.WriteJSON(edit); err != nil {
		t.Fatalf("send edit: %v", err)
	}

	if !readSystemReplyContaining(t, conn, "empty content", time.Second) {
		t.Fatal("expected System reply about empty content")
	}

	var content string
	if err := db.QueryRow(`SELECT content FROM messages WHERE message_id = ?`, msgID).Scan(&content); err != nil {
		t.Fatalf("query: %v", err)
	}
	if content != "keep-me" {
		t.Fatalf("edit must not replace content, got %q", content)
	}
}

func TestIntegrationEncryptedOpaqueTextAccepted(t *testing.T) {
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer db.Close()
	CreateSchema(db)

	hub := NewHub(tdir, tdir, "", db)
	go hub.Run()

	handler := ServeWs(hub, db, nil, "admin-key", false, 10<<20, dbPath)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(shared.Handshake{Username: "emptyEnc1"}); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	const blob = "bm9uY2UxMjM0NTY3ODkwYWJjZGVmZ2hpamtsbW5vcA=="
	msg := shared.Message{
		Content:   blob,
		Type:      shared.TextMessage,
		Encrypted: true,
	}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("send: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	var gotEcho bool
	for {
		var got shared.Message
		if err := conn.ReadJSON(&got); err != nil {
			break
		}
		if got.Sender == "System" && strings.Contains(got.Content, "empty content") {
			t.Fatal("encrypted opaque content must not be rejected as empty")
		}
		if got.Content == blob && got.Encrypted {
			gotEcho = true
			break
		}
	}
	if !gotEcho {
		t.Fatal("expected encrypted message echo/broadcast")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM messages WHERE content = ?`, blob).Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Fatalf("encrypted opaque content should be persisted, got %d rows", count)
	}
}

func TestIntegrationCommandPathStillWorksWithEmptyCheck(t *testing.T) {
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer db.Close()
	CreateSchema(db)

	hub := NewHub(tdir, tdir, "", db)
	go hub.Run()

	handler := ServeWs(hub, db, nil, "admin-key", false, 10<<20, dbPath)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(shared.Handshake{Username: "emptyCmd1"}); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	cmd := shared.Message{
		Content: ":hello",
		Type:    shared.TextMessage,
	}
	if err := conn.WriteJSON(cmd); err != nil {
		t.Fatalf("send: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	for {
		var got shared.Message
		if err := conn.ReadJSON(&got); err != nil {
			t.Fatal("expected System reply from command path")
		}
		if got.Sender == "System" && strings.Contains(got.Content, "empty content") {
			t.Fatal("command path must not hit empty-content reject")
		}
		if got.Sender == "System" && strings.TrimSpace(got.Content) != "" {
			return
		}
	}
}
