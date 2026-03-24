package shared

import (
	"strings"
	"testing"
)

func TestGetVersionInfo(t *testing.T) {
	// Test with default values
	version := GetVersionInfo()

	// Should contain client version, build time, and git commit
	if !strings.Contains(version, ClientVersion) {
		t.Errorf("Version info should contain client version %s, got: %s", ClientVersion, version)
	}

	if !strings.Contains(version, BuildTime) {
		t.Errorf("Version info should contain build time %s, got: %s", BuildTime, version)
	}

	if !strings.Contains(version, GitCommit) {
		t.Errorf("Version info should contain git commit %s, got: %s", GitCommit, version)
	}

	// Should have proper format with parentheses
	if !strings.Contains(version, "(") || !strings.Contains(version, ")") {
		t.Errorf("Version info should contain parentheses, got: %s", version)
	}

	// Should have commas separating the parts
	if !strings.Contains(version, ",") {
		t.Errorf("Version info should contain commas, got: %s", version)
	}
}

func TestGetServerVersionInfo(t *testing.T) {
	// Test with default values
	version := GetServerVersionInfo()

	// Should contain server version, build time, and git commit
	if !strings.Contains(version, ServerVersion) {
		t.Errorf("Server version info should contain server version %s, got: %s", ServerVersion, version)
	}

	if !strings.Contains(version, BuildTime) {
		t.Errorf("Server version info should contain build time %s, got: %s", BuildTime, version)
	}

	if !strings.Contains(version, GitCommit) {
		t.Errorf("Server version info should contain git commit %s, got: %s", GitCommit, version)
	}

	// Should have proper format with parentheses
	if !strings.Contains(version, "(") || !strings.Contains(version, ")") {
		t.Errorf("Server version info should contain parentheses, got: %s", version)
	}

	// Should have commas separating the parts
	if !strings.Contains(version, ",") {
		t.Errorf("Server version info should contain commas, got: %s", version)
	}
}

func TestVersionVariables(t *testing.T) {
	// Test that version variables are set to expected default values
	if ClientVersion != "dev" {
		t.Errorf("Expected ClientVersion to be 'dev', got: %s", ClientVersion)
	}

	if ServerVersion != "dev" {
		t.Errorf("Expected ServerVersion to be 'dev', got: %s", ServerVersion)
	}

	if BuildTime != "unknown" {
		t.Errorf("Expected BuildTime to be 'unknown', got: %s", BuildTime)
	}

	if GitCommit != "unknown" {
		t.Errorf("Expected GitCommit to be 'unknown', got: %s", GitCommit)
	}
}

func TestVersionInfoConsistency(t *testing.T) {
	// Test that both version functions return consistent format
	clientVersion := GetVersionInfo()
	serverVersion := GetServerVersionInfo()

	// Both should have the same format structure
	clientParts := strings.Split(clientVersion, " ")
	serverParts := strings.Split(serverVersion, " ")

	if len(clientParts) != len(serverParts) {
		t.Errorf("Version info formats should be consistent. Client: %s, Server: %s", clientVersion, serverVersion)
	}

	// Both should contain build and commit info
	if !strings.Contains(clientVersion, "build:") || !strings.Contains(clientVersion, "commit:") {
		t.Errorf("Client version should contain 'build:' and 'commit:', got: %s", clientVersion)
	}

	if !strings.Contains(serverVersion, "build:") || !strings.Contains(serverVersion, "commit:") {
		t.Errorf("Server version should contain 'build:' and 'commit:', got: %s", serverVersion)
	}
}

func TestVersionInfoNonEmpty(t *testing.T) {
	// Test that version functions return non-empty strings
	clientVersion := GetVersionInfo()
	serverVersion := GetServerVersionInfo()

	if clientVersion == "" {
		t.Error("Client version info should not be empty")
	}

	if serverVersion == "" {
		t.Error("Server version info should not be empty")
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{name: "equal", a: "1.2.3", b: "1.2.3", want: 0},
		{name: "less_than_major", a: "1.0.0", b: "2.0.0", want: -1},
		{name: "greater_than_major", a: "2.0.0", b: "1.0.0", want: 1},
		{name: "less_than_minor", a: "1.2.0", b: "1.3.0", want: -1},
		{name: "less_than_patch", a: "1.2.3", b: "1.2.4", want: -1},
		{name: "dev_before_release", a: "dev", b: "1.0.0", want: -1},
		{name: "v_prefix_equal", a: "v1.2.3", b: "1.2.3", want: 0},
		{name: "two_part_equal_to_three", a: "1.2", b: "1.2.0", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareVersions(tt.a, tt.b); got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %d; want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
