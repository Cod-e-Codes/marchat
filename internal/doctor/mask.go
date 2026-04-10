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
		return fmt.Sprintf("(set, len=%d)", len(value))
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

// ClientHookMarchatEnvKeys lists client-only hook-related variables appended after KnownMarchatEnvKeys
// when running client doctor. Server doctor omits these rows (they are not read by the server).
// MARCHAT_HOOK_LOG is consumed by the bundled example_hook binary, not the marchat client itself.
var ClientHookMarchatEnvKeys = []string{
	"MARCHAT_CLIENT_HOOK_RECEIVE",
	"MARCHAT_CLIENT_HOOK_SEND",
	"MARCHAT_CLIENT_HOOK_TIMEOUT_SEC",
	"MARCHAT_CLIENT_HOOK_RECEIVE_TYPING",
	"MARCHAT_CLIENT_HOOK_DEBUG",
	"MARCHAT_HOOK_LOG",
}
