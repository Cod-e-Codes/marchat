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

func TestPluginContentWithinLimit(t *testing.T) {
	limit := int64(8)
	if !pluginContentWithinLimit("12345678", limit) {
		t.Fatal("body at the limit must be kept")
	}
	if pluginContentWithinLimit("123456789", limit) {
		t.Fatal("body over the limit must be dropped")
	}
	if !pluginContentWithinLimit("hi", 0) {
		t.Fatal("short plugin body must fit the default cap")
	}
}

func TestDispatchInboundMessageSizeLimit(t *testing.T) {
	client, _ := setupDispatchTestClient(t)
	client.maxMessageBytes = 8

	oversize := strings.Repeat("x", 9)
	cases := []shared.Message{
		{Type: shared.TextMessage, Content: oversize},
		{Type: shared.DirectMessage, Recipient: "other", Content: oversize},
		{Type: shared.EditMessageType, MessageID: 1, Content: oversize},
		{Type: shared.SearchMessage, Content: oversize},
		{Type: shared.AdminCommandType, Content: ":" + oversize},
		{Type: shared.TextMessage, Content: oversize, Encrypted: true},
	}
	for _, msg := range cases {
		t.Run(string(msg.Type)+encryptedSuffix(msg.Encrypted), func(t *testing.T) {
			drainSend(client.send)
			client.dispatchInbound(&msg)
			if !waitSystemSend(t, client.send, "exceeds maximum size limit", time.Second) {
				t.Fatal("expected System reply about message size")
			}
			if msg.Encrypted {
				got := false
				select {
				case v := <-client.send:
					if m, ok := v.(shared.Message); ok && strings.Contains(m.Content, "empty content") {
						got = true
					}
				default:
				}
				if got {
					t.Fatal("encrypted oversize must not be treated as empty content")
				}
			}
		})
	}

	var count int
	if err := client.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE content = ?`, oversize).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("oversize content persisted %d rows", count)
	}

	t.Run("at limit accepted", func(t *testing.T) {
		drainSend(client.send)
		body := strings.Repeat("y", 8)
		msg := shared.Message{Type: shared.TextMessage, Content: body}
		client.dispatchInbound(&msg)
		select {
		case v := <-client.send:
			m, ok := v.(shared.Message)
			if !ok {
				t.Fatalf("unexpected payload %T", v)
			}
			if strings.Contains(m.Content, "exceeds maximum size") {
				t.Fatalf("at-limit content rejected: %s", m.Content)
			}
			if m.Content != body {
				t.Fatalf("broadcast content = %q, want %q", m.Content, body)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for broadcast of at-limit message")
		}
		if err := client.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE content = ?`, body).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("at-limit content count = %d, want 1", count)
		}
	})
}

func encryptedSuffix(encrypted bool) string {
	if encrypted {
		return "_encrypted"
	}
	return ""
}

func TestIntegrationMessageTooLargeKeepsConnection(t *testing.T) {
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer CloseDB(db)
	CreateSchema(db)

	hub := mustNewHub(t, tdir, tdir, "", db)
	hub.SetMaxMessageBytes(16)
	go hub.Run()

	const maxMessage int64 = 16
	handler := ServeWs(hub, db, nil, "admin-key", false, 10<<20, maxMessage, dbPath)
	srv := httptest.NewServer(handler)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(shared.Handshake{Username: "sizeuser"}); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	oversize := strings.Repeat("z", int(maxMessage)+1)
	if err := conn.WriteJSON(shared.Message{Type: shared.TextMessage, Content: oversize}); err != nil {
		t.Fatalf("send oversize: %v", err)
	}
	if !readSystemReplyContaining(t, conn, "exceeds maximum size limit", 2*time.Second) {
		t.Fatal("expected System reply about message size")
	}

	var stored int
	if err := db.QueryRow(`SELECT COUNT(*) FROM messages WHERE content = ?`, oversize).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != 0 {
		t.Fatalf("oversize content persisted %d rows", stored)
	}

	const follow = "still-open"
	if err := conn.WriteJSON(shared.Message{Type: shared.TextMessage, Content: follow}); err != nil {
		t.Fatalf("connection not writable after reject: %v", err)
	}
	if !readContentContaining(t, conn, follow, 2*time.Second) {
		t.Fatal("did not receive follow-up chat after size reject")
	}
}

func readContentContaining(t *testing.T, conn *websocket.Conn, substr string, timeout time.Duration) bool {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	for {
		var msg shared.Message
		if err := conn.ReadJSON(&msg); err != nil {
			return false
		}
		if strings.Contains(msg.Content, substr) {
			return true
		}
	}
}
