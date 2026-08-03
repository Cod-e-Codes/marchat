package server

import (
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

func TestFileMessageReadLimit(t *testing.T) {
	tests := []struct {
		name         string
		maxFileBytes int64
	}{
		{"1kb", 1024},
		{"1mb", 1024 * 1024},
		{"default_when_zero", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			limit := fileMessageReadLimit(tc.maxFileBytes)
			maxRaw := tc.maxFileBytes
			if maxRaw <= 0 {
				maxRaw = 1024 * 1024
			}
			data := make([]byte, maxRaw)
			msg := shared.Message{
				Type: shared.FileMessageType,
				File: &shared.FileMeta{
					Filename: "max.bin",
					Size:     maxRaw,
					Data:     data,
				},
			}
			wire, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("marshal max file message: %v", err)
			}
			if int64(len(wire)) > limit {
				t.Fatalf("read limit %d is smaller than max allowed wire size %d", limit, len(wire))
			}

			oversized := make([]byte, maxRaw+maxRaw)
			msg.File.Data = oversized
			msg.File.Size = int64(len(oversized))
			wire, err = json.Marshal(msg)
			if err != nil {
				t.Fatalf("marshal oversized file message: %v", err)
			}
			if int64(len(wire)) <= limit {
				t.Fatalf("oversized wire size %d should exceed read limit %d", len(wire), limit)
			}
		})
	}
}

func setupFileLimitHub(t *testing.T, maxFileBytes int64) (string, func()) {
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

	handler := ServeWs(hub, db, nil, "admin-key", false, maxFileBytes, dbPath)
	srv := httptest.NewServer(handler)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	cleanup := func() {
		srv.Close()
		db.Close()
	}
	return wsURL, cleanup
}

func readNextSystemMessage(conn *websocket.Conn, timeout time.Duration) (shared.Message, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		var msg shared.Message
		if err := conn.ReadJSON(&msg); err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return shared.Message{}, err
		}
		if msg.Sender == "System" {
			return msg, nil
		}
	}
	return shared.Message{}, errTimeoutWaitingSystem
}

var errTimeoutWaitingSystem = &timeoutError{"timeout waiting for system message"}

type timeoutError struct{ msg string }

func (e *timeoutError) Error() string { return e.msg }

func TestIntegrationOversizedFileDeclaredSizeRejected(t *testing.T) {
	const maxFile = int64(256)
	wsURL, cleanup := setupFileLimitHub(t, maxFile)
	defer cleanup()

	conn := dialWS(t, wsURL, "file-declarer")
	defer conn.Close()
	time.Sleep(150 * time.Millisecond)

	bad := shared.Message{
		Type: shared.FileMessageType,
		File: &shared.FileMeta{
			Filename: "small.bin",
			Size:     maxFile + 500,
			Data:     []byte("tiny"),
		},
	}
	if err := conn.WriteJSON(bad); err != nil {
		t.Fatalf("send declared oversized file: %v", err)
	}

	sys, err := readNextSystemMessage(conn, 2*time.Second)
	if err != nil {
		t.Fatalf("expected system reply: %v", err)
	}
	if !strings.Contains(sys.Content, "File not sent") || !strings.Contains(sys.Content, "maximum size limit") {
		t.Fatalf("unexpected system content: %q", sys.Content)
	}

	// Connection should stay usable after a header-only rejection.
	ok := shared.Message{Content: "still-here", Type: shared.TextMessage}
	if err := conn.WriteJSON(ok); err != nil {
		t.Fatalf("send follow-up after rejection: %v", err)
	}
}

func TestIntegrationOversizedFilePayloadRejected(t *testing.T) {
	const maxFile = int64(256)
	wsURL, cleanup := setupFileLimitHub(t, maxFile)
	defer cleanup()

	conn := dialWS(t, wsURL, "file-payload")
	defer conn.Close()
	time.Sleep(150 * time.Millisecond)

	data := make([]byte, maxFile+1)
	bad := shared.Message{
		Type: shared.FileMessageType,
		File: &shared.FileMeta{
			Filename: "lie.bin",
			Size:     10,
			Data:     data,
		},
	}
	if err := conn.WriteJSON(bad); err != nil {
		t.Fatalf("send oversized payload with small declared size: %v", err)
	}

	sys, err := readNextSystemMessage(conn, 2*time.Second)
	if err != nil {
		t.Fatalf("expected system reply: %v", err)
	}
	if !strings.Contains(sys.Content, "File not sent") {
		t.Fatalf("unexpected system content: %q", sys.Content)
	}
}

func TestIntegrationOversizedFileWireRejected(t *testing.T) {
	const maxFile = int64(256)
	wsURL, cleanup := setupFileLimitHub(t, maxFile)
	defer cleanup()

	conn := dialWS(t, wsURL, "file-wire")
	defer conn.Close()
	time.Sleep(150 * time.Millisecond)

	data := make([]byte, 4096)
	bad := shared.Message{
		Type: shared.FileMessageType,
		File: &shared.FileMeta{
			Filename: "big.bin",
			Size:     int64(len(data)),
			Data:     data,
		},
	}
	if err := conn.WriteJSON(bad); err != nil {
		t.Fatalf("send oversized wire file: %v", err)
	}

	// gorilla may close with 1009 before a system reply is delivered; either is acceptable.
	sys, sysErr := readNextSystemMessage(conn, 2*time.Second)
	if sysErr == nil {
		if !strings.Contains(sys.Content, "File not sent") {
			t.Fatalf("unexpected system content: %q", sys.Content)
		}
		return
	}

	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _, readErr := conn.ReadMessage()
	if readErr == nil {
		t.Fatal("expected connection close or system message for oversized wire file")
	}
	if !websocket.IsCloseError(readErr, websocket.CloseMessageTooBig) {
		t.Fatalf("expected close 1009, got: %v", readErr)
	}
}
