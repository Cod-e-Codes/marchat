package doctor

import (
	"strings"
	"testing"
)

func TestFormatEnvValueRedaction(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		want    string
		notWant string // substring that must NOT appear in output
	}{
		{
			name:  "empty value",
			key:   "MARCHAT_PORT",
			value: "",
			want:  "(not set)",
		},
		{
			name:  "non-secret value",
			key:   "MARCHAT_PORT",
			value: "8080",
			want:  "8080",
		},
		{
			name:  "long non-secret value truncated",
			key:   "MARCHAT_PORT",
			value: strings.Repeat("a", 200),
			want:  "...",
		},
		{
			name:    "secret fully redacted",
			key:     "MARCHAT_ADMIN_KEY",
			value:   "super-secret-key-1234",
			want:    "(set, len=21)",
			notWant: "1234",
		},
		{
			name:    "session secret fully redacted",
			key:     "MARCHAT_SESSION_SECRET",
			value:   "abcdef1234567890",
			want:    "(set, len=16)",
			notWant: "7890",
		},
		{
			name:    "e2e key fully redacted",
			key:     "MARCHAT_GLOBAL_E2E_KEY",
			value:   "base64encodedkey==",
			want:    "(set, len=18)",
			notWant: "y==",
		},
		{
			name:  "short secret fully redacted",
			key:   "MARCHAT_ADMIN_KEY",
			value: "ab",
			want:  "(set, len=2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatEnvValue(tt.key, tt.value)
			if !strings.Contains(got, tt.want) {
				t.Errorf("FormatEnvValue(%q, %q) = %q, want substring %q", tt.key, tt.value, got, tt.want)
			}
			if tt.notWant != "" && strings.Contains(got, tt.notWant) {
				t.Errorf("FormatEnvValue(%q, %q) = %q, must NOT contain %q", tt.key, tt.value, got, tt.notWant)
			}
		})
	}
}

func TestIsSecretEnv(t *testing.T) {
	secrets := []string{
		"MARCHAT_ADMIN_KEY",
		"MARCHAT_SESSION_SECRET",
		"MARCHAT_JWT_SECRET",
		"MARCHAT_GLOBAL_E2E_KEY",
	}
	for _, key := range secrets {
		if !isSecretEnv(key) {
			t.Errorf("expected %q to be secret", key)
		}
	}

	nonSecrets := []string{
		"MARCHAT_PORT",
		"MARCHAT_DB_PATH",
		"MARCHAT_LOG_LEVEL",
	}
	for _, key := range nonSecrets {
		if isSecretEnv(key) {
			t.Errorf("expected %q to NOT be secret", key)
		}
	}
}
