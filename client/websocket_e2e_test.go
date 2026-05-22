package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/client/crypto"
	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/gorilla/websocket"
)

func TestBuildEncryptedOutboundMessageDM(t *testing.T) {
	ks := crypto.NewKeyStore(t.TempDir() + "/keystore.dat")
	if err := ks.Initialize("test-passphrase"); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	msg, err := buildEncryptedOutboundMessage(ks, "alice", "hello dm", shared.DirectMessage, "bob")
	if err != nil {
		t.Fatalf("buildEncryptedOutboundMessage: %v", err)
	}
	if msg.Type != shared.DirectMessage {
		t.Errorf("type = %q, want %q", msg.Type, shared.DirectMessage)
	}
	if msg.Recipient != "bob" {
		t.Errorf("recipient = %q, want bob", msg.Recipient)
	}
	if !msg.Encrypted {
		t.Error("expected encrypted true")
	}
	if msg.Content == "" || msg.Content == "hello dm" {
		t.Error("content should be opaque ciphertext, not plaintext")
	}
	if msg.Sender != "alice" {
		t.Errorf("sender = %q, want alice", msg.Sender)
	}
}

func TestBuildEncryptedOutboundMessageChannel(t *testing.T) {
	ks := crypto.NewKeyStore(t.TempDir() + "/keystore.dat")
	if err := ks.Initialize("test-passphrase"); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	msg, err := buildEncryptedOutboundMessage(ks, "alice", "hello channel", shared.TextMessage, "")
	if err != nil {
		t.Fatalf("buildEncryptedOutboundMessage: %v", err)
	}
	if msg.Type != shared.TextMessage {
		t.Errorf("type = %q, want %q", msg.Type, shared.TextMessage)
	}
	if msg.Recipient != "" {
		t.Errorf("recipient = %q, want empty", msg.Recipient)
	}
	if !msg.Encrypted {
		t.Error("expected encrypted true")
	}
}

func TestSendDirectMessageEncryptedWire(t *testing.T) {
	ks := crypto.NewKeyStore(t.TempDir() + "/keystore.dat")
	if err := ks.Initialize("test-passphrase"); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		var got shared.Message
		if err := conn.ReadJSON(&got); err != nil {
			t.Errorf("ReadJSON: %v", err)
			return
		}
		if got.Type != shared.DirectMessage {
			t.Errorf("type = %q, want dm", got.Type)
		}
		if got.Recipient != "bob" {
			t.Errorf("recipient = %q, want bob", got.Recipient)
		}
		if !got.Encrypted {
			t.Error("expected encrypted dm on wire")
		}
		if got.Content == "" || strings.Contains(got.Content, "secret") {
			t.Error("wire content should be ciphertext, not plaintext")
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	if err := sendDirectMessage(conn, ks, "alice", "bob", "secret", true); err != nil {
		t.Fatalf("sendDirectMessage: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
}

func TestSendDirectMessagePlaintext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		var got shared.Message
		if err := conn.ReadJSON(&got); err != nil {
			t.Errorf("ReadJSON: %v", err)
			return
		}
		if got.Encrypted {
			t.Error("plaintext dm should not set encrypted")
		}
		if got.Content != "hi" {
			t.Errorf("content = %q, want hi", got.Content)
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	if err := sendDirectMessage(conn, nil, "alice", "bob", "hi", false); err != nil {
		t.Fatalf("sendDirectMessage: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
}

func TestEncryptedDMRoundtripDecrypt(t *testing.T) {
	ks := crypto.NewKeyStore(t.TempDir() + "/keystore.dat")
	if err := ks.Initialize("test-passphrase"); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	out, err := buildEncryptedOutboundMessage(ks, "alice", "dm body", shared.DirectMessage, "bob")
	if err != nil {
		t.Fatalf("buildEncryptedOutboundMessage: %v", err)
	}

	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var chatMsg shared.Message
	if err := json.Unmarshal(raw, &chatMsg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !chatMsg.Encrypted || chatMsg.Type != shared.DirectMessage {
		t.Fatalf("unexpected message shape: %+v", chatMsg)
	}

	decrypted, err := decryptEncryptedChatContent(ks, chatMsg)
	if err != nil {
		t.Fatalf("decryptEncryptedChatContent: %v", err)
	}
	if decrypted != "dm body" {
		t.Errorf("decrypted = %q, want dm body", decrypted)
	}
}
