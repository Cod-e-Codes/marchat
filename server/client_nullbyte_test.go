package server

import (
	"database/sql"
	"encoding/json"
	"net"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/gorilla/websocket"
)

func TestContentContainsNUL(t *testing.T) {
	if contentContainsNUL("hello") {
		t.Fatal("plain text should not contain NUL")
	}
	if !contentContainsNUL("before\x00after") {
		t.Fatal("expected NUL detection")
	}
}

func TestInsertMessage_NullByteAcceptedOnSQLite(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	CreateSchema(db)

	content := "before\x00after"
	id, err := InsertMessage(db, shared.Message{
		Sender:    "user",
		Content:   content,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("sqlite insert with null byte failed: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}
}

func drainWSJSONUntilIdle(conn *websocket.Conn, idle time.Duration) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(idle))
		var raw json.RawMessage
		if err := conn.ReadJSON(&raw); err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return
			}
			return
		}
	}
}

func TestIntegrationNullByteRejectedNoBroadcast(t *testing.T) {
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

	listener := dial("listener1")
	defer listener.Close()
	drainWSJSONUntilIdle(listener, 200*time.Millisecond)

	sender := dial("sender99")
	defer sender.Close()
	drainWSJSONUntilIdle(sender, 200*time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	drainWSJSONUntilIdle(listener, 200*time.Millisecond)

	const content = "nul-byte-test"
	bad := shared.Message{
		Content: content + "\x00" + "tail",
		Type:    shared.TextMessage,
	}
	if err := sender.WriteJSON(bad); err != nil {
		t.Fatalf("send: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	rows, err := db.Query(`SELECT content FROM messages WHERE content LIKE ?`, content+"%")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("NUL message must not be persisted")
	}

	_ = listener.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
	var peerMsg shared.Message
	if err := listener.ReadJSON(&peerMsg); err == nil && strings.Contains(peerMsg.Content, content) {
		t.Fatal("peer should not receive broadcast for rejected NUL message")
	}
}
