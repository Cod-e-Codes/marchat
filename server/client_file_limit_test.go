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

func TestWebsocketReadLimit(t *testing.T) {
	smallPolicy := int64(100 << 10) // 100KiB: real-world style small config
	got := websocketReadLimit(smallPolicy)
	if got != maxWebSocketMessageBytes {
		t.Fatalf("small policy: websocketReadLimit=%d, want DoS ceiling %d", got, maxWebSocketMessageBytes)
	}
	policyWire := fileMessageReadLimit(smallPolicy)
	if got <= policyWire {
		t.Fatalf("websocketReadLimit %d must exceed policy wire %d so modest oversize is readable", got, policyWire)
	}

	// Just-over-policy file must fit under the DoS ceiling (post-parse reject path).
	overshoot := smallPolicy + 1024
	overMsg := shared.Message{
		Type: shared.FileMessageType,
		File: &shared.FileMeta{
			Filename: "over.bin",
			Size:     overshoot,
			Data:     make([]byte, overshoot),
		},
	}
	wire, err := json.Marshal(overMsg)
	if err != nil {
		t.Fatalf("marshal overshoot: %v", err)
	}
	if int64(len(wire)) <= policyWire {
		t.Fatalf("overshoot wire %d should exceed policy wire %d", len(wire), policyWire)
	}
	if int64(len(wire)) > got {
		t.Fatalf("overshoot wire %d must be under websocketReadLimit %d", len(wire), got)
	}

	zeroDefault := websocketReadLimit(0)
	if zeroDefault != maxWebSocketMessageBytes {
		t.Fatalf("default policy: websocketReadLimit=%d, want %d", zeroDefault, maxWebSocketMessageBytes)
	}

	// Policy wire above the DoS ceiling: honor max allowed file size.
	hugeMax := int64(40 << 20) // 40MiB raw -> policy wire > 32MiB
	hugePolicy := fileMessageReadLimit(hugeMax)
	if hugePolicy <= maxWebSocketMessageBytes {
		t.Fatalf("test setup: expected policy wire %d > ceiling %d", hugePolicy, maxWebSocketMessageBytes)
	}
	if got := websocketReadLimit(hugeMax); got != hugePolicy {
		t.Fatalf("huge policy: websocketReadLimit=%d, want policy wire %d", got, hugePolicy)
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

	hub := mustNewHub(t, tdir, tdir, "", db)
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

// assertSystemTextFrameBeforeClose reads at the opcode level and requires a text
// frame whose JSON contains "File not sent" before any close frame. This mirrors
// raw-client observation better than ReadJSON alone.
func assertSystemTextFrameBeforeClose(t *testing.T, conn *websocket.Conn, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if websocket.IsCloseError(err, websocket.CloseMessageTooBig, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				t.Fatalf("got close frame before System text frame: %v", err)
			}
			t.Fatalf("read frame: %v", err)
		}
		switch msgType {
		case websocket.TextMessage, websocket.BinaryMessage:
			if !strings.Contains(string(payload), "File not sent") {
				// Handshake replay or other chatter; keep scanning.
				continue
			}
			var msg shared.Message
			if err := json.Unmarshal(payload, &msg); err != nil {
				t.Fatalf("unmarshal System text frame: %v (payload %q)", err, payload)
			}
			if msg.Sender != "System" {
				continue
			}
			return
		case websocket.CloseMessage:
			t.Fatalf("got close opcode before System text frame (payload %q)", payload)
		default:
			// Ping/pong/continuation: ignore.
			continue
		}
	}
	t.Fatal("timeout waiting for System text frame before close")
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
	wire, err := json.Marshal(bad)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if int64(len(wire)) <= fileMessageReadLimit(maxFile) {
		t.Fatalf("test setup: wire %d should exceed policy limit %d", len(wire), fileMessageReadLimit(maxFile))
	}
	if int64(len(wire)) > websocketReadLimit(maxFile) {
		t.Fatalf("test setup: wire %d must fit under DoS ceiling %d", len(wire), websocketReadLimit(maxFile))
	}

	if err := conn.WriteJSON(bad); err != nil {
		t.Fatalf("send oversized wire file: %v", err)
	}

	assertSystemTextFrameBeforeClose(t, conn, 2*time.Second)

	// Connection should stay usable after app-layer rejection.
	ok := shared.Message{Content: "still-here", Type: shared.TextMessage}
	if err := conn.WriteJSON(ok); err != nil {
		t.Fatalf("send follow-up after rejection: %v", err)
	}
}

func TestIntegrationOversizedFileSmallPolicyWireRejected(t *testing.T) {
	// Closest to the #114 repro class: small MARCHAT_MAX_FILE_BYTES, file just
	// over policy, wire well under the 32MiB DoS ceiling.
	const maxFile = int64(100 << 10) // 100KiB
	wsURL, cleanup := setupFileLimitHub(t, maxFile)
	defer cleanup()

	conn := dialWS(t, wsURL, "file-small-policy")
	defer conn.Close()
	time.Sleep(150 * time.Millisecond)

	overshoot := maxFile + 2048
	bad := shared.Message{
		Type: shared.FileMessageType,
		File: &shared.FileMeta{
			Filename: "just-over.bin",
			Size:     overshoot,
			Data:     make([]byte, overshoot),
		},
	}
	wire, err := json.Marshal(bad)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	policy := fileMessageReadLimit(maxFile)
	dos := websocketReadLimit(maxFile)
	if int64(len(wire)) <= policy {
		t.Fatalf("test setup: wire %d should exceed policy %d", len(wire), policy)
	}
	if int64(len(wire)) > dos {
		t.Fatalf("test setup: wire %d must fit under DoS ceiling %d", len(wire), dos)
	}

	if err := conn.WriteJSON(bad); err != nil {
		t.Fatalf("send just-over-policy file: %v", err)
	}

	assertSystemTextFrameBeforeClose(t, conn, 2*time.Second)

	ok := shared.Message{Content: "still-here", Type: shared.TextMessage}
	if err := conn.WriteJSON(ok); err != nil {
		t.Fatalf("send follow-up after rejection: %v", err)
	}
}
