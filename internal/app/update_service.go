package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/SabirDzh/VpnCLI/internal/updater"
)

// UpdateService checks for and installs CLI updates.
type UpdateService struct {
	Repo    string
	Current string
	Client  *http.Client
	// DataDir holds the update record file; empty disables recording.
	DataDir string

	mu     sync.Mutex
	record updateRecord
}

type updateRecord struct {
	LastCheck   time.Time `json:"lastCheck"`
	LastUpdated time.Time `json:"lastUpdated"`
}

// NewUpdateService wires the service.
func NewUpdateService(repo, current, dataDir string) *UpdateService {
	return &UpdateService{
		Repo: repo, Current: current, DataDir: dataDir,
		Client: &http.Client{Timeout: 60 * time.Second},
	}
}

// CheckResult describes the update state.
type CheckResult struct {
	Current   string
	Latest    string
	Available bool
}

// Check compares the current version against the latest release.
// "dev" builds always report Available=false with Latest resolved.
func (s *UpdateService) Check(ctx context.Context) (CheckResult, error) {
	latest, err := updater.LatestTag(ctx, s.Client, s.Repo)
	if err != nil {
		return CheckResult{}, err
	}
	res := CheckResult{Current: s.Current, Latest: latest}
	if s.Current != "" && s.Current != "dev" {
		res.Available = updater.Compare(s.Current, latest) < 0
	}
	s.recordCheck()
	return res, nil
}

// AutoCheck checks and, when a newer release exists, installs it.
// Returns the installed tag or "" when already up to date. "dev" builds
// never self-update.
func (s *UpdateService) AutoCheck(ctx context.Context) (string, error) {
	res, err := s.Check(ctx)
	if err != nil {
		return "", err
	}
	if !res.Available {
		return "", nil
	}
	if err := s.Update(ctx, res.Latest); err != nil {
		return "", err
	}
	return res.Latest, nil
}

func (s *UpdateService) recordPath() string {
	if s.DataDir == "" {
		return ""
	}
	return filepath.Join(s.DataDir, "updates.json")
}

func (s *UpdateService) loadRecordLocked() {
	s.record = updateRecord{}
	data, err := os.ReadFile(s.recordPath())
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &s.record)
}

func (s *UpdateService) saveRecordLocked() {
	path := s.recordPath()
	if path == "" {
		return
	}
	data, err := json.MarshalIndent(&s.record, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, data, 0o600)
}

func (s *UpdateService) recordCheck() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadRecordLocked()
	s.record.LastCheck = time.Now()
	s.saveRecordLocked()
}

// LastCheck reports when updates were last checked (zero if never).
func (s *UpdateService) LastCheck() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadRecordLocked()
	return s.record.LastCheck
}

// LastUpdated reports when an update was last installed (zero if never).
func (s *UpdateService) LastUpdated() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadRecordLocked()
	return s.record.LastUpdated
}

// Update downloads the release asset for this platform and replaces
// the current executable.
func (s *UpdateService) Update(ctx context.Context, tag string) error {
	if tag == "" {
		latest, err := updater.LatestTag(ctx, s.Client, s.Repo)
		if err != nil {
			return err
		}
		tag = latest
	}
	url := updater.DownloadURL(s.Repo, tag, runtime.GOOS, runtime.GOARCH)
	dst, err := updater.DownloadAndExtract(ctx, s.Client, url, runtime.GOOS)
	if err != nil {
		return err
	}
	defer os.Remove(dst)
	if err := replaceExecutable(dst); err != nil {
		return err
	}
	s.mu.Lock()
	s.loadRecordLocked()
	s.record.LastUpdated = time.Now()
	s.saveRecordLocked()
	s.mu.Unlock()
	return nil
}

func replaceExecutable(src string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// src is our own MkdirTemp download (gosec exclusion, see line below).
	in, err := os.Open(src) //nolint:gosec
	if err != nil {
		return err
	}
	defer in.Close()
	// exe is our own binary path (gosec exclusion, see line below).
	out, err := os.OpenFile(exe, os.O_WRONLY|os.O_TRUNC, 0o755) //nolint:gosec
	if err != nil {
		return fmt.Errorf("replace %s (re-run with rights to write it): %w", exe, err)
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
