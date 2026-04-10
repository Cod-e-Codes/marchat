// Package exthook runs optional external processes for experimental client-side automation.
// See CLIENT_HOOKS.md in the repo root for the full protocol and security notes.
// Hook-related MARCHAT_* variables appear in marchat-client -doctor output; receive/send paths are validated there when set.
//
// Enable with environment variables (no config file yet):
//
//	MARCHAT_CLIENT_HOOK_RECEIVE: absolute path to executable (one JSON line on stdin)
//	MARCHAT_CLIENT_HOOK_SEND: same for outbound chat-related sends
//	MARCHAT_CLIENT_HOOK_TIMEOUT_SEC: optional, default 5
//	MARCHAT_CLIENT_HOOK_RECEIVE_TYPING=1: include typing indicators in receive hook (default: off)
//	MARCHAT_CLIENT_HOOK_DEBUG=1: log successful hook runs (duration, optional stdout preview)
//
// Security: only absolute paths are accepted. Hooks see plaintext after decrypt (receive)
// or plaintext before wire send (send). Do not point at untrusted binaries.
package exthook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
)

const payloadVersion = 1

// Event names are stable for trial integrations; not yet a public protocol.
const (
	EventMessageReceived = "message_received"
	EventMessageSend     = "message_send"
)

var (
	loadOnce sync.Once
	cfg      config
)

type config struct {
	receivePath   string
	sendPath      string
	timeout       time.Duration
	receiveTyping bool
	debugSuccess  bool
}

func envTruthy(key string) bool {
	s := strings.TrimSpace(os.Getenv(key))
	return s == "1" || strings.EqualFold(s, "true") || strings.EqualFold(s, "yes")
}

func loadConfig() {
	loadOnce.Do(func() {
		cfg.timeout = 5 * time.Second
		if s := strings.TrimSpace(os.Getenv("MARCHAT_CLIENT_HOOK_TIMEOUT_SEC")); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 120 {
				cfg.timeout = time.Duration(n) * time.Second
			}
		}
		cfg.receivePath = strings.TrimSpace(os.Getenv("MARCHAT_CLIENT_HOOK_RECEIVE"))
		cfg.sendPath = strings.TrimSpace(os.Getenv("MARCHAT_CLIENT_HOOK_SEND"))
		cfg.receiveTyping = envTruthy("MARCHAT_CLIENT_HOOK_RECEIVE_TYPING")
		cfg.debugSuccess = envTruthy("MARCHAT_CLIENT_HOOK_DEBUG")
	})
}

func validateExecutable(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("hook path must be absolute: %q", path)
	}
	clean := filepath.Clean(path)
	st, err := os.Stat(clean)
	if err != nil {
		return "", err
	}
	if st.IsDir() {
		return "", fmt.Errorf("hook path is a directory: %q", clean)
	}
	// Windows may run .bat via Cmd; require regular file to avoid surprises.
	if !st.Mode().IsRegular() {
		return "", fmt.Errorf("hook path must be a regular file: %q", clean)
	}
	return clean, nil
}

// ValidateHookExecutable checks that path is non-empty, absolute, and refers to an existing regular file.
// It returns the cleaned path. Used by client diagnostics (-doctor).
func ValidateHookExecutable(path string) (string, error) {
	return validateExecutable(strings.TrimSpace(path))
}

// messageForHook returns a JSON-serializable view; file bytes are never included.
func messageForHook(msg shared.Message) map[string]any {
	out := map[string]any{
		"type":       msg.Type,
		"sender":     msg.Sender,
		"content":    msg.Content,
		"encrypted":  msg.Encrypted,
		"message_id": msg.MessageID,
		"recipient":  msg.Recipient,
		"edited":     msg.Edited,
		"channel":    msg.Channel,
	}
	if !msg.CreatedAt.IsZero() {
		out["created_at"] = msg.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	if msg.Reaction != nil {
		out["reaction"] = msg.Reaction
	}
	if msg.File != nil {
		out["file"] = map[string]any{
			"filename": msg.File.Filename,
			"size":     msg.File.Size,
		}
	}
	return out
}

// FireReceive runs asynchronously after the client has prepared a chat message (including decrypt).
func FireReceive(msg shared.Message) {
	loadConfig()
	if msg.Type == shared.TypingMessage && !cfg.receiveTyping {
		return
	}
	path, err := validateExecutable(cfg.receivePath)
	if err != nil {
		return
	}
	go runHook(path, EventMessageReceived, msg)
}

// FireSend runs asynchronously before the corresponding wire send (plaintext).
func FireSend(msg shared.Message) {
	loadConfig()
	path, err := validateExecutable(cfg.sendPath)
	if err != nil {
		return
	}
	go runHook(path, EventMessageSend, msg)
}

func runHook(execPath, event string, msg shared.Message) {
	payload := map[string]any{
		"event":   event,
		"version": payloadVersion,
		"message": messageForHook(msg),
	}
	line, err := json.Marshal(payload)
	if err != nil {
		log.Printf("exthook: marshal: %v", err)
		return
	}
	line = append(line, '\n')

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(ctx, execPath)
	cmd.Stdin = bytes.NewReader(line)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	out, err := cmd.Output()
	elapsed := time.Since(start)
	if ctx.Err() == context.DeadlineExceeded {
		log.Printf("exthook: %s %q timed out after %v", event, execPath, cfg.timeout)
		return
	}
	if err != nil {
		log.Printf("exthook: %s %q failed: %v", event, execPath, err)
		if stderrBuf.Len() > 0 {
			log.Printf("exthook: stderr: %s", strings.TrimSpace(stderrBuf.String()))
		}
		return
	}
	if cfg.debugSuccess {
		log.Printf("exthook: %s %q ok in %v", event, execPath, elapsed)
	}
	if len(bytes.TrimSpace(out)) > 0 {
		const maxLog = 4096
		s := string(out)
		if len(s) > maxLog {
			s = s[:maxLog] + "..."
		}
		preview := strings.TrimSpace(s)
		if cfg.debugSuccess {
			log.Printf("exthook: %s stdout preview: %s", event, preview)
		} else {
			log.Printf("exthook: %s stdout: %s", event, preview)
		}
	}
}
