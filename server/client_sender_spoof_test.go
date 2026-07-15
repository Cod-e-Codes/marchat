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

func setupSpoofTestHub(t *testing.T) (*sql.DB, string, func()) {
	t.Helper()
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	CreateSchema(db)

	hub := NewHub(tdir, tdir, "", db)
	go hub.Run()

	handler := ServeWs(hub, db, nil, "admin-key", false, 10<<20, dbPath)
	srv := httptest.NewServer(handler)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	cleanup := func() {
		srv.Close()
		db.Close()
	}
	return db, wsURL, cleanup
}

func dialWS(t *testing.T, wsURL, username string) *websocket.Conn {
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

func drainUntilQuiet(conn *websocket.Conn, idle time.Duration) {
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

func TestIntegrationPlainTextIgnoresSpoofedSender(t *testing.T) {
	db, wsURL, cleanup := setupSpoofTestHub(t)
	defer cleanup()

	spoofer := dialWS(t, wsURL, "realuser99")
	defer spoofer.Close()
	drainUntilQuiet(spoofer, 200*time.Millisecond)

	const content = "sender-spoof-poc"
	spoofed := shared.Message{
		Sender:  "admin1",
		Content: content,
		Type:    shared.TextMessage,
	}
	if err := spoofer.WriteJSON(spoofed); err != nil {
		t.Fatalf("send spoofed message: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	rows, err := db.Query(`SELECT sender, content FROM messages WHERE content = ?`, content)
	if err != nil {
		t.Fatalf("query db: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("message was not persisted")
	}
	var dbSender, dbContent string
	if err := rows.Scan(&dbSender, &dbContent); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if dbSender != "realuser99" {
		t.Fatalf("db sender: got %q want realuser99", dbSender)
	}
}

func TestIntegrationFileMessageIgnoresSpoofedSender(t *testing.T) {
	client, hub, _, cleanup := setupTestClient(t)
	defer cleanup()
	hub.joinChannel(client, "general")

	msg := shared.Message{
		Sender: "admin1",
		Type:   shared.FileMessageType,
		File: &shared.FileMeta{
			Filename: "note.txt",
			Size:     4,
			Data:     []byte("data"),
		},
	}
	client.stampSenderTimedOutbound(&msg)
	if msg.Sender != "testuser" {
		t.Fatalf("file message sender: got %q want testuser", msg.Sender)
	}
	if msg.Channel != "general" {
		t.Fatalf("file message channel: got %q want general", msg.Channel)
	}
}

func TestStampSenderTimedOutboundOverwritesSpoofedSender(t *testing.T) {
	client, hub, _, cleanup := setupTestClient(t)
	defer cleanup()
	hub.joinChannel(client, "general")

	msg := shared.Message{Sender: "admin1", Content: "hi", Type: shared.TextMessage}
	client.stampSenderTimedOutbound(&msg)
	if msg.Sender != "testuser" {
		t.Fatalf("stampSenderTimedOutbound: got %q want testuser", msg.Sender)
	}
}

func TestStampTimedOutboundDoesNotOverwriteSender(t *testing.T) {
	client, hub, _, cleanup := setupTestClient(t)
	defer cleanup()
	hub.joinChannel(client, "general")

	msg := shared.Message{Sender: "admin1", Content: "hi", Type: shared.TextMessage}
	client.stampTimedOutbound(&msg)
	if msg.Sender != "admin1" {
		t.Fatalf("stampTimedOutbound unexpectedly overwrote sender: %q", msg.Sender)
	}
}
