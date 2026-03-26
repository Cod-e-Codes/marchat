package doctor

import "fmt"

var secretEnvKeys = map[string]struct{}{
	"MARCHAT_ADMIN_KEY":      {},
	"MARCHAT_SESSION_SECRET": {},
	"MARCHAT_JWT_SECRET":     {},
	"MARCHAT_GLOBAL_E2E_KEY": {},
}

func isSecretEnv(key string) bool {
	_, ok := secretEnvKeys[key]
	return ok
}

// FormatEnvValue returns a human-safe display string for an environment value.
func FormatEnvValue(key, value string) string {
	if value == "" {
		return "(not set)"
	}
	if isSecretEnv(key) {
		n := len(value)
		suf := ""
		if n >= 4 {
			suf = value[n-4:]
		}
		return fmt.Sprintf("(set, len=%d, suffix=****%s)", n, suf)
	}
	if len(value) > 120 {
		return value[:117] + "..."
	}
	return value
}

// KnownMarchatEnvKeys lists documented MARCHAT_* variables for doctor output.
var KnownMarchatEnvKeys = []string{
	"MARCHAT_CONFIG_DIR",
	"MARCHAT_PORT",
	"MARCHAT_ADMIN_KEY",
	"MARCHAT_USERS",
	"MARCHAT_DB_PATH",
	"MARCHAT_LOG_LEVEL",
	"MARCHAT_SESSION_SECRET",
	"MARCHAT_JWT_SECRET",
	"MARCHAT_TLS_CERT_FILE",
	"MARCHAT_TLS_KEY_FILE",
	"MARCHAT_BAN_HISTORY_GAPS",
	"MARCHAT_PLUGIN_REGISTRY_URL",
	"MARCHAT_MAX_FILE_BYTES",
	"MARCHAT_MAX_FILE_MB",
	"MARCHAT_GLOBAL_E2E_KEY",
	"MARCHAT_ALLOWED_USERS",
	"MARCHAT_DOCTOR_NO_NETWORK",
}
