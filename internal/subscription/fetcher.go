// Package subscription fetches and parses subscription lists.
package subscription

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/subscription/uri"
)

const (
	maxBody    = 2 << 20 // 2 MiB
	userAgent  = "vpn-cli/1.0"
	httpTimout = 15 * time.Second
)

// Fetch downloads a subscription body and returns parsed profiles.
// Supports plain URI lists and base64-encoded lists.
func Fetch(url string) ([]domain.Profile, error) {
	client := &http.Client{Timeout: httpTimout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch subscription: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch subscription: http %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBody {
		return nil, fmt.Errorf("subscription body too large")
	}
	return ParseContent(strings.TrimSpace(string(body)))
}

// ParseContent handles base64-wrapped or plain subscription bodies.
func ParseContent(body string) ([]domain.Profile, error) {
	if body == "" {
		return nil, nil
	}
	if !strings.Contains(body, "://") {
		if decoded, err := base64.StdEncoding.DecodeString(cleanB64(body)); err == nil {
			body = string(decoded)
		} else if decoded, err := base64.RawStdEncoding.DecodeString(cleanB64(body)); err == nil {
			body = string(decoded)
		}
	}
	var out []domain.Profile
	sc := bufio.NewScanner(strings.NewReader(body))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, err := uri.Parse(line)
		if err != nil {
			continue // skip unsupported lines, keep the rest
		}
		out = append(out, p)
	}
	return out, sc.Err()
}

func cleanB64(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}
