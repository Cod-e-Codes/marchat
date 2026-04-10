package exthook

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
)

func TestValidateExecutable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	exe := filepath.Join(dir, "hook")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\necho\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		t.Fatal(err)
	}
	got, err := validateExecutable(abs)
	if err != nil {
		t.Fatalf("valid file: %v", err)
	}
	if got != abs {
		t.Fatalf("got %q want %q", got, abs)
	}
	if _, err := validateExecutable("relative/hook"); err == nil {
		t.Fatal("expected error for relative path")
	}
	if _, err := validateExecutable(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMessageForHookOmitsFileBytes(t *testing.T) {
	t.Parallel()
	msg := shared.Message{
		Sender:    "a",
		Content:   "",
		Type:      shared.FileMessageType,
		CreatedAt: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		File: &shared.FileMeta{
			Filename: "x.bin",
			Size:     3,
			Data:     []byte{1, 2, 3},
		},
	}
	m := messageForHook(msg)
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, []byte(`"data"`)) || strings.Contains(string(b), "AQID") {
		t.Fatalf("file bytes leaked into hook payload: %s", string(b))
	}
}

func TestValidateHookExecutable(t *testing.T) {
	t.Parallel()
	if _, err := ValidateHookExecutable(""); err == nil {
		t.Fatal("empty path should error")
	}
	if _, err := ValidateHookExecutable("relative/hook"); err == nil {
		t.Fatal("relative path should error")
	}
}

func TestMessageForHookOmitsZeroCreatedAt(t *testing.T) {
	t.Parallel()
	m := messageForHook(shared.Message{
		Sender:  "u",
		Content: "hi",
		Type:    shared.TextMessage,
	})
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "created_at") {
		t.Fatalf("expected zero CreatedAt to omit created_at, got %s", string(raw))
	}
}
