package server

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
)

func setupDispatchTestClient(t *testing.T) (*Client, *Hub) {
	t.Helper()

	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	CreateSchema(db)

	hub := mustNewHub(t, tdir, tdir, "", db)
	go hub.Run()

	client := &Client{
		hub:                  hub,
		send:                 make(chan interface{}, 64),
		db:                   db,
		username:             "dispatchuser",
		maxFileBytes:         1024,
		pluginCommandHandler: hub.pluginCommandHandler,
	}
	hub.clientsMutex.Lock()
	hub.clients[client] = true
	hub.clientsByUsername[strings.ToLower(client.username)] = client
	hub.clientsMutex.Unlock()
	hub.joinChannel(client, "general")

	return client, hub
}

func drainSend(ch chan interface{}) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func waitSystemSend(t *testing.T, ch chan interface{}, substr string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			return false
		case v := <-ch:
			msg, ok := v.(shared.Message)
			if !ok {
				continue
			}
			if msg.Sender == "System" && strings.Contains(msg.Content, substr) {
				return true
			}
		}
	}
}

func TestDispatchInboundRoutesNoPanic(t *testing.T) {
	client, _ := setupDispatchTestClient(t)

	msgID, err := InsertMessage(client.db, shared.Message{
		Sender:    client.username,
		Content:   "seed for edit/delete",
		CreatedAt: time.Now(),
		Type:      shared.TextMessage,
	})
	if err != nil {
		t.Fatalf("InsertMessage: %v", err)
	}

	cases := []struct {
		name string
		msg  shared.Message
	}{
		{name: "file", msg: shared.Message{
			Type: shared.FileMessageType,
			File: &shared.FileMeta{Filename: "a.txt", Size: 3, Data: []byte("abc")},
		}},
		{name: "file_too_large", msg: shared.Message{
			Type: shared.FileMessageType,
			File: &shared.FileMeta{Filename: "big.bin", Size: 2048, Data: make([]byte, 2048)},
		}},
		{name: "edit", msg: shared.Message{
			Type: shared.EditMessageType, MessageID: msgID, Content: "edited body",
		}},
		{name: "delete", msg: shared.Message{
			Type: shared.DeleteMessage, MessageID: msgID,
		}},
		{name: "typing_channel", msg: shared.Message{Type: shared.TypingMessage}},
		{name: "typing_dm", msg: shared.Message{
			Type: shared.TypingMessage, Recipient: "dmpeer",
		}},
		{name: "reaction", msg: shared.Message{
			Type: shared.ReactionMessage,
			Reaction: &shared.ReactionMeta{
				Emoji: "thumbsup", TargetID: msgID,
			},
		}},
		{name: "dm", msg: shared.Message{
			Type: shared.DirectMessage, Recipient: "dmpeer", Content: "hello dm",
		}},
		{name: "dm_empty", msg: shared.Message{
			Type: shared.DirectMessage, Recipient: "dmpeer", Content: "   ",
		}},
		{name: "search", msg: shared.Message{
			Type: shared.SearchMessage, Content: "seed",
		}},
		{name: "pin_list", msg: shared.Message{Type: shared.PinMessage}},
		{name: "pin_toggle_non_admin", msg: shared.Message{
			Type: shared.PinMessage, MessageID: msgID,
		}},
		{name: "read_receipt", msg: shared.Message{
			Type: shared.ReadReceiptType, MessageID: msgID,
		}},
		{name: "join_channel", msg: shared.Message{
			Type: shared.JoinChannelType, Channel: "random",
		}},
		{name: "leave_channel", msg: shared.Message{Type: shared.LeaveChannelType}},
		{name: "list_channels", msg: shared.Message{Type: shared.ListChannelsType}},
		{name: "command", msg: shared.Message{Content: ":stats"}},
		{name: "admin_command_type", msg: shared.Message{
			Type: shared.AdminCommandType, Content: ":stats",
		}},
		{name: "text", msg: shared.Message{
			Type: shared.TextMessage, Content: "hello world",
		}},
		{name: "text_empty", msg: shared.Message{
			Type: shared.TextMessage, Content: "",
		}},
		{name: "edit_skipped_zero_id", msg: shared.Message{
			Type: shared.EditMessageType, Content: "orphan edit",
		}},
		{name: "file_nil_meta", msg: shared.Message{Type: shared.FileMessageType}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			drainSend(client.send)
			msg := tc.msg
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("dispatchInbound panicked: %v", r)
				}
			}()
			client.dispatchInbound(&msg)
		})
	}
}

func TestDispatchInboundSystemMessages(t *testing.T) {
	client, _ := setupDispatchTestClient(t)

	t.Run("empty_text", func(t *testing.T) {
		drainSend(client.send)
		msg := shared.Message{Type: shared.TextMessage, Content: "   "}
		client.dispatchInbound(&msg)
		if !waitSystemSend(t, client.send, "empty content", time.Second) {
			t.Fatal("expected System reply about empty content")
		}
	})

	t.Run("empty_dm", func(t *testing.T) {
		drainSend(client.send)
		msg := shared.Message{
			Type: shared.DirectMessage, Recipient: "someone", Content: "",
		}
		client.dispatchInbound(&msg)
		if !waitSystemSend(t, client.send, "empty content", time.Second) {
			t.Fatal("expected System reply about empty DM content")
		}
	})

	t.Run("file_too_large", func(t *testing.T) {
		drainSend(client.send)
		msg := shared.Message{
			Type: shared.FileMessageType,
			File: &shared.FileMeta{
				Filename: "huge.bin",
				Size:     2048,
				Data:     make([]byte, 2048),
			},
		}
		client.dispatchInbound(&msg)
		if !waitSystemSend(t, client.send, "exceeds maximum size limit", time.Second) {
			t.Fatal("expected System reply about file size")
		}
	})

	t.Run("pin_non_admin", func(t *testing.T) {
		msgID, err := InsertMessage(client.db, shared.Message{
			Sender: client.username, Content: "pin me", CreatedAt: time.Now(),
		})
		if err != nil {
			t.Fatalf("InsertMessage: %v", err)
		}
		drainSend(client.send)
		msg := shared.Message{Type: shared.PinMessage, MessageID: msgID}
		client.dispatchInbound(&msg)
		if !waitSystemSend(t, client.send, "Only admins can pin messages", time.Second) {
			t.Fatal("expected System reply denying pin")
		}
	})

	t.Run("join_channel", func(t *testing.T) {
		drainSend(client.send)
		msg := shared.Message{Type: shared.JoinChannelType, Channel: "ops"}
		client.dispatchInbound(&msg)
		if !waitSystemSend(t, client.send, "Joined channel #ops", time.Second) {
			t.Fatal("expected System join confirmation")
		}
	})

	t.Run("search_no_results", func(t *testing.T) {
		drainSend(client.send)
		msg := shared.Message{
			Type: shared.SearchMessage, Content: "zzznomatchzzzz",
		}
		client.dispatchInbound(&msg)
		if !waitSystemSend(t, client.send, "No results found for:", time.Second) {
			t.Fatal("expected System search empty reply")
		}
	})

	t.Run("pin_list_empty", func(t *testing.T) {
		drainSend(client.send)
		msg := shared.Message{Type: shared.PinMessage}
		client.dispatchInbound(&msg)
		if !waitSystemSend(t, client.send, "No pinned messages", time.Second) {
			t.Fatal("expected System pinned list empty reply")
		}
	})

	t.Run("list_channels", func(t *testing.T) {
		drainSend(client.send)
		msg := shared.Message{Type: shared.ListChannelsType}
		client.dispatchInbound(&msg)
		if !waitSystemSend(t, client.send, "Active channels:", time.Second) {
			t.Fatal("expected System channel list reply")
		}
	})
}
