package main

import (
	"strings"
	"testing"

	"github.com/Cod-e-Codes/marchat/shared"
)

func TestBuildStatusFooter(t *testing.T) {
	tests := []struct {
		name            string
		connected       bool
		showHelp        bool
		unread          int
		useE2E          bool
		currentChannel  string
		activeDMThread  string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:            "connected_plain",
			connected:       true,
			showHelp:        false,
			unread:          0,
			useE2E:          false,
			currentChannel:  "general",
			activeDMThread:  "",
			wantContains:    []string{"Connected"},
			wantNotContains: []string{"Unread", "E2E", "#general", "Ctrl+H", "Unencrypted", "Msg info"},
		},
		{
			name:            "disconnected_shows_help",
			connected:       false,
			showHelp:        false,
			unread:          0,
			useE2E:          false,
			currentChannel:  "",
			activeDMThread:  "",
			wantContains:    []string{"Disconnected", "Press Ctrl+H for help"},
			wantNotContains: []string{"Msg info"},
		},
		{
			name:            "help_open_connected",
			connected:       true,
			showHelp:        true,
			unread:          0,
			useE2E:          false,
			currentChannel:  "general",
			activeDMThread:  "",
			wantContains:    []string{"Connected", "Press Ctrl+H to close help"},
			wantNotContains: []string{"Press Ctrl+H for help"},
		},
		{
			name:            "unread_e2e_channel",
			connected:       true,
			showHelp:        false,
			unread:          3,
			useE2E:          true,
			currentChannel:  "dev",
			activeDMThread:  "",
			wantContains:    []string{"Connected", "3 unread", "E2E", "#dev"},
			wantNotContains: []string{"Unencrypted", "Msg info"},
		},
		{
			name:            "dm_thread",
			connected:       true,
			showHelp:        false,
			unread:          0,
			useE2E:          false,
			currentChannel:  "general",
			activeDMThread:  "alice",
			wantContains:    []string{"Connected", "DM:alice"},
			wantNotContains: []string{"#general"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildStatusFooter(tt.connected, tt.showHelp, tt.unread, tt.useE2E, tt.currentChannel, tt.activeDMThread)
			for _, s := range tt.wantContains {
				if !strings.Contains(got, s) {
					t.Errorf("footer %q should contain %q", got, s)
				}
			}
			for _, s := range tt.wantNotContains {
				if strings.Contains(got, s) {
					t.Errorf("footer %q should not contain %q", got, s)
				}
			}
		})
	}
}

func TestStripKindForBanner(t *testing.T) {
	tests := []struct {
		text string
		want bannerStripKind
	}{
		{"", bannerStripInfo},
		{"[OK] Connected", bannerStripInfo},
		{"Msg info: full", bannerStripInfo},
		{"  [WARN] clipboard", bannerStripWarn},
		{"[WARN] Connection lost", bannerStripWarn},
		{"[ERROR] failed", bannerStripError},
		{"[ERROR] x [Sending...]", bannerStripError},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := stripKindForBanner(tt.text); got != tt.want {
				t.Fatalf("stripKindForBanner(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestLayoutBannerForStrip(t *testing.T) {
	long := "[ERROR] Failed to read file: open " + strings.Repeat("x", 200) + ": no such file"
	out := layoutBannerForStrip(long, 80)
	if strings.Contains(out, "\n") {
		t.Fatal("banner layout must be single line")
	}
	if len([]rune(out)) > 80 {
		t.Fatalf("expected truncation under width, got len %d", len([]rune(out)))
	}
	if !strings.HasSuffix(out, "...") {
		t.Fatal("expected ellipsis for long banner")
	}
}

func TestMaxMessageID(t *testing.T) {
	msgs := []shared.Message{
		{MessageID: 10},
		{MessageID: 2},
		{MessageID: 99},
	}
	if id := maxMessageID(msgs); id != 99 {
		t.Fatalf("maxMessageID = %d, want 99", id)
	}
	if id := maxMessageID(nil); id != 0 {
		t.Fatalf("maxMessageID(nil) = %d, want 0", id)
	}
}
