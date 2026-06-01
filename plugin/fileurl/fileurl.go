package fileurl

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// Parse converts a file:// URL string to a native filesystem path.
func Parse(raw string) (string, error) {
	if strings.HasPrefix(raw, "file://") {
		rest := strings.TrimPrefix(raw, "file://")
		if strings.ContainsRune(rest, '\\') {
			raw = "file://" + filepath.ToSlash(rest)
		}
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid file URL: %w", err)
	}
	return Path(parsed)
}

// Path converts a parsed file URL to a native filesystem path.
func Path(parsed *url.URL) (string, error) {
	if parsed == nil || parsed.Scheme != "file" {
		return "", fmt.Errorf("not a file URL")
	}

	host := parsed.Host
	if host == "localhost" {
		host = ""
	}

	var filePath string
	switch {
	case host == "":
		var err error
		filePath, err = url.PathUnescape(parsed.Path)
		if err != nil {
			return "", fmt.Errorf("invalid file URL path: %w", err)
		}
	case len(host) == 2 && host[1] == ':':
		// file://C:/path on Windows (host is the drive letter)
		filePath = host + parsed.Path
	default:
		return "", fmt.Errorf("unsupported file URL host %q", parsed.Host)
	}

	if filePath == "" {
		return "", fmt.Errorf("file URL path cannot be empty")
	}
	if len(filePath) >= 3 && filePath[0] == '/' && filePath[2] == ':' {
		filePath = filePath[1:]
	}
	return filepath.FromSlash(filePath), nil
}
