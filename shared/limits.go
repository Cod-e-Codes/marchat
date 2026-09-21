package shared

import (
	"fmt"
	"os"
	"strconv"
)

const (
	// DefaultMaxMessageBytes is the chat content cap when neither message env var is set.
	DefaultMaxMessageBytes int64 = 32 << 10
	// DefaultMaxFileBytes is the file payload cap when neither file env var is set.
	DefaultMaxFileBytes int64 = 1024 * 1024

	EnvMaxMessageBytes = "MARCHAT_MAX_MESSAGE_BYTES"
	EnvMaxMessageMB    = "MARCHAT_MAX_MESSAGE_MB"
	EnvMaxFileBytes    = "MARCHAT_MAX_FILE_BYTES"
	EnvMaxFileMB       = "MARCHAT_MAX_FILE_MB"
)

// EffectiveMaxMessageBytes returns n, or DefaultMaxMessageBytes when n is not positive.
func EffectiveMaxMessageBytes(n int64) int64 {
	if n <= 0 {
		return DefaultMaxMessageBytes
	}
	return n
}

// ContentExceedsLimit reports whether content's UTF-8 byte length is above the
// effective message cap. A length equal to the cap is allowed.
func ContentExceedsLimit(content string, maxBytes int64) bool {
	return int64(len(content)) > EffectiveMaxMessageBytes(maxBytes)
}

// ParseMaxBytesEnv reads bytesKey, then mbKey (megabytes). An empty pair returns
// defaultBytes. A present value that does not parse or is not positive is an error.
// bytesKey wins when both are set.
func ParseMaxBytesEnv(bytesKey, mbKey string, defaultBytes int64) (int64, error) {
	const oneMB int64 = 1024 * 1024
	if bytesStr := os.Getenv(bytesKey); bytesStr != "" {
		val, err := strconv.ParseInt(bytesStr, 10, 64)
		if err != nil || val <= 0 {
			return 0, fmt.Errorf("invalid %s: %s", bytesKey, bytesStr)
		}
		return val, nil
	}
	if mbStr := os.Getenv(mbKey); mbStr != "" {
		val, err := strconv.ParseInt(mbStr, 10, 64)
		if err != nil || val <= 0 {
			return 0, fmt.Errorf("invalid %s: %s", mbKey, mbStr)
		}
		return val * oneMB, nil
	}
	if defaultBytes <= 0 {
		return 0, fmt.Errorf("invalid default byte limit")
	}
	return defaultBytes, nil
}

// MaxBytesFromEnv is the client path: an invalid or non-positive setting falls
// back to defaultBytes instead of failing startup.
func MaxBytesFromEnv(bytesKey, mbKey string, defaultBytes int64) int64 {
	n, err := ParseMaxBytesEnv(bytesKey, mbKey, defaultBytes)
	if err != nil || n <= 0 {
		if defaultBytes <= 0 {
			return DefaultMaxMessageBytes
		}
		return defaultBytes
	}
	return n
}

// FormatMessageLimit renders a message cap for admin UI. Non-positive n uses the default.
// Whole megabytes use the same "%.1f MB" form as file size. Other whole kibibytes use KiB.
func FormatMessageLimit(n int64) string {
	n = EffectiveMaxMessageBytes(n)
	const oneMB int64 = 1024 * 1024
	const oneKiB int64 = 1024
	if n%oneMB == 0 {
		return fmt.Sprintf("%.1f MB", float64(n)/float64(oneMB))
	}
	if n%oneKiB == 0 {
		return fmt.Sprintf("%d KiB", n/oneKiB)
	}
	return fmt.Sprintf("%d bytes", n)
}
