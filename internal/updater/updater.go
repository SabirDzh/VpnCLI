// Package updater checks GitHub releases for newer versions and
// self-updates the binary from release assets.
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Release is the subset of GitHub release metadata we need.
type Release struct {
	Tag string `json:"tag_name"`
}

// apiBase is overridable in tests.
var apiBase = "https://api.github.com"

// LatestTag returns the latest release tag (e.g. "v0.1.1").
func LatestTag(ctx context.Context, client *http.Client, repo string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, "GET",
		apiBase+"/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("check updates: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("check updates: http %s", resp.Status)
	}
	var r Release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", fmt.Errorf("decode release: %w", err)
	}
	if r.Tag == "" {
		return "", fmt.Errorf("check updates: empty tag")
	}
	return r.Tag, nil
}

// Compare returns -1/0/+1 for a vs b ("v" prefix tolerated).
func Compare(a, b string) int {
	pa := split(strings.TrimPrefix(a, "v"))
	pb := split(strings.TrimPrefix(b, "v"))
	for i := 0; i < 3; i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

// AssetName mirrors the goreleaser name_template (version without "v").
func AssetName(tag, goos, goarch string) string {
	ver := strings.TrimPrefix(tag, "v")
	if goos == "windows" {
		return fmt.Sprintf("vpn_%s_windows_%s.zip", ver, goarch)
	}
	return fmt.Sprintf("vpn_%s_%s_%s.tar.gz", ver, goos, goarch)
}

// DownloadURL builds the release asset URL.
func DownloadURL(repo, tag, goos, goarch string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s",
		repo, tag, AssetName(tag, goos, goarch))
}

func split(v string) []string { return strings.Split(v, ".") }
