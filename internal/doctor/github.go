package doctor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const githubLatestReleasesURL = "https://api.github.com/repos/Cod-e-Codes/marchat/releases/latest"

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// ParseLatestReleaseTag extracts tag_name from a GitHub releases/latest JSON body (for tests).
func ParseLatestReleaseTag(body []byte) (string, error) {
	var rel githubRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return "", err
	}
	rel.TagName = strings.TrimSpace(rel.TagName)
	if rel.TagName == "" {
		return "", fmt.Errorf("empty tag_name")
	}
	return rel.TagName, nil
}

func fetchLatestReleaseTag(client *http.Client) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequest(http.MethodGet, githubLatestReleasesURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "marchat-doctor")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	return ParseLatestReleaseTag(body)
}
