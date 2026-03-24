package shared

import (
	"fmt"
	"strconv"
	"strings"
)

// Version variables that can be set at build time using ldflags
var (
	// ClientVersion is the version of the MarChat client
	ClientVersion = "dev"

	// ServerVersion is the version of the MarChat server
	ServerVersion = "dev"

	// BuildTime is the time when the binary was built
	BuildTime = "unknown"

	// GitCommit is the git commit hash
	GitCommit = "unknown"
)

// GetVersionInfo returns a formatted version string
func GetVersionInfo() string {
	return fmt.Sprintf("%s (build: %s, commit: %s)", ClientVersion, BuildTime, GitCommit)
}

// GetServerVersionInfo returns a formatted server version string
func GetServerVersionInfo() string {
	return fmt.Sprintf("%s (build: %s, commit: %s)", ServerVersion, BuildTime, GitCommit)
}

// CompareVersions compares two semver-style version strings (e.g. "1.2.3").
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Non-numeric or "dev" versions are treated as 0.0.0 so they satisfy any constraint.
func CompareVersions(a, b string) int {
	pa := parseVersionParts(a)
	pb := parseVersionParts(b)
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

func parseVersionParts(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return [3]int{}
		}
		result[i] = n
	}
	return result
}
