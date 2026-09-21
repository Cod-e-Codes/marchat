package shared

import (
	"strings"
	"testing"
)

func TestEffectiveMaxMessageBytes(t *testing.T) {
	if got := EffectiveMaxMessageBytes(0); got != DefaultMaxMessageBytes {
		t.Fatalf("zero: got %d, want %d", got, DefaultMaxMessageBytes)
	}
	if got := EffectiveMaxMessageBytes(-1); got != DefaultMaxMessageBytes {
		t.Fatalf("negative: got %d, want %d", got, DefaultMaxMessageBytes)
	}
	if got := EffectiveMaxMessageBytes(64); got != 64 {
		t.Fatalf("explicit: got %d, want 64", got)
	}
}

func TestContentExceedsLimit(t *testing.T) {
	const max int64 = 4
	if ContentExceedsLimit("abcd", max) {
		t.Fatal("length equal to the cap must be allowed")
	}
	if !ContentExceedsLimit("abcde", max) {
		t.Fatal("length above the cap must be rejected")
	}
	if ContentExceedsLimit("hi", 0) {
		t.Fatal("short content must fit the default cap")
	}
}

func TestParseMaxBytesEnv(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv(EnvMaxMessageBytes, "")
		t.Setenv(EnvMaxMessageMB, "")
		got, err := ParseMaxBytesEnv(EnvMaxMessageBytes, EnvMaxMessageMB, DefaultMaxMessageBytes)
		if err != nil {
			t.Fatal(err)
		}
		if got != DefaultMaxMessageBytes {
			t.Fatalf("got %d, want %d", got, DefaultMaxMessageBytes)
		}
	})

	t.Run("bytes win over mb", func(t *testing.T) {
		t.Setenv(EnvMaxMessageBytes, "100")
		t.Setenv(EnvMaxMessageMB, "2")
		got, err := ParseMaxBytesEnv(EnvMaxMessageBytes, EnvMaxMessageMB, DefaultMaxMessageBytes)
		if err != nil {
			t.Fatal(err)
		}
		if got != 100 {
			t.Fatalf("got %d, want 100", got)
		}
	})

	t.Run("mb", func(t *testing.T) {
		t.Setenv(EnvMaxMessageBytes, "")
		t.Setenv(EnvMaxMessageMB, "2")
		got, err := ParseMaxBytesEnv(EnvMaxMessageBytes, EnvMaxMessageMB, DefaultMaxMessageBytes)
		if err != nil {
			t.Fatal(err)
		}
		if got != 2*1024*1024 {
			t.Fatalf("got %d, want 2MB", got)
		}
	})

	t.Run("invalid bytes", func(t *testing.T) {
		t.Setenv(EnvMaxMessageBytes, "0")
		t.Setenv(EnvMaxMessageMB, "")
		if _, err := ParseMaxBytesEnv(EnvMaxMessageBytes, EnvMaxMessageMB, DefaultMaxMessageBytes); err == nil {
			t.Fatal("expected error for non-positive bytes")
		}
	})

	t.Run("invalid mb", func(t *testing.T) {
		t.Setenv(EnvMaxFileBytes, "")
		t.Setenv(EnvMaxFileMB, "nope")
		_, err := ParseMaxBytesEnv(EnvMaxFileBytes, EnvMaxFileMB, DefaultMaxFileBytes)
		if err == nil || !strings.Contains(err.Error(), "invalid "+EnvMaxFileMB) {
			t.Fatalf("got %v", err)
		}
	})
}

func TestMaxBytesFromEnvFallsBack(t *testing.T) {
	t.Setenv(EnvMaxMessageBytes, "-5")
	t.Setenv(EnvMaxMessageMB, "")
	got := MaxBytesFromEnv(EnvMaxMessageBytes, EnvMaxMessageMB, DefaultMaxMessageBytes)
	if got != DefaultMaxMessageBytes {
		t.Fatalf("got %d, want default %d", got, DefaultMaxMessageBytes)
	}
}

func TestFormatMessageLimit(t *testing.T) {
	if got := FormatMessageLimit(0); got != "32 KiB" {
		t.Fatalf("default: got %q, want 32 KiB", got)
	}
	if got := FormatMessageLimit(DefaultMaxMessageBytes); got != "32 KiB" {
		t.Fatalf("32KiB: got %q", got)
	}
	if got := FormatMessageLimit(1024 * 1024); got != "1.0 MB" {
		t.Fatalf("1MB: got %q", got)
	}
	if got := FormatMessageLimit(100); got != "100 bytes" {
		t.Fatalf("bytes: got %q", got)
	}
}
