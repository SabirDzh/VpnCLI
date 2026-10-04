package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// StatsAPI contract shared with the TUI lives in shared (StatsAPI); this
// service implements it on top of sing-box's clash_api plus the log file.
type StatsService struct {
	base    string
	logFile string
	client  *http.Client
}

// Traffic is a totals snapshot.
type Traffic struct {
	Up      int64 // bytes sent through the core
	Down    int64 // bytes received
	Active  int   // live connections
	Blocked int   // reject rule hits found in the log
}

// NewStatsService targets the clash_api controller (host:port) and log file.
func NewStatsService(clashBase, logFile string) *StatsService {
	return &StatsService{
		base:    "http://" + clashBase,
		logFile: logFile,
		client:  &http.Client{Timeout: 2 * time.Second},
	}
}

// connectionsResp is the subset of clash_api /connections used here.
type connectionsResp struct {
	UploadTotal   int64 `json:"uploadTotal"`
	DownloadTotal int64 `json:"downloadTotal"`
	Connections   []any `json:"connections"`
}

// Totals returns cumulative traffic and live connection counts.
func (s *StatsService) Totals(ctx context.Context) (Traffic, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.base+"/connections", nil)
	if err != nil {
		return Traffic{}, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return Traffic{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Traffic{}, fmt.Errorf("clash api: %s", resp.Status)
	}
	var c connectionsResp
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return Traffic{}, err
	}
	blocked, _ := s.Blocked()
	return Traffic{Up: c.UploadTotal, Down: c.DownloadTotal, Active: len(c.Connections), Blocked: blocked}, nil
}

var rejectRe = regexp.MustCompile(`(?i)reject`)

// Blocked counts reject hits in the log file. sing-box logs every routed
// rejection, so this approximates the total number of blocked requests
// since the log file was last truncated (per VPN run).
func (s *StatsService) Blocked() (int, error) {
	raw, err := os.ReadFile(s.logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	n := 0
	for _, line := range strings.Split(string(raw), "\n") {
		if rejectRe.MatchString(line) {
			n++
		}
	}
	return n, nil
}

// TailLog returns up to n most recent log lines (oldest first).
func (s *StatsService) TailLog(n int) []string {
	raw, err := os.ReadFile(s.logFile)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}
