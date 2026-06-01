package fileurl

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseUnixAbsolutePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix path form")
	}
	got, err := Parse("file:///tmp/registry.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := filepath.FromSlash("/tmp/registry.json")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestParseLocalhostHost(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix path form")
	}
	got, err := Parse("file://localhost/tmp/registry.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := filepath.FromSlash("/tmp/registry.json")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestParseWindowsDriveInHost(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows file URL form")
	}
	got, err := Parse("file://C:/Users/test/registry.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := filepath.FromSlash("C:/Users/test/registry.json")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestParseWindowsThreeSlashDrivePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows file URL form")
	}
	got, err := Parse("file:///C:/Users/test/registry.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := filepath.FromSlash("C:/Users/test/registry.json")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path, got %q", got)
	}
}

func TestParseWindowsBackslashPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows file URL form")
	}
	got, err := Parse(`file://C:\Users\test\registry.json`)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := filepath.FromSlash("C:/Users/test/registry.json")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestParseRejectsUnsupportedHost(t *testing.T) {
	_, err := Parse("file://remotehost/share/registry.json")
	if err == nil {
		t.Fatal("expected unsupported host error")
	}
}
