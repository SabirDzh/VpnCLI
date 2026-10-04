package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/SabirDzh/VpnCLI/internal/updater"
)

// UpdateService checks for and installs CLI updates.
type UpdateService struct {
	Repo    string
	Current string
	Client  *http.Client
}

// NewUpdateService wires the service.
func NewUpdateService(repo, current string) *UpdateService {
	return &UpdateService{
		Repo: repo, Current: current,
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
	if s.Current == "" || s.Current == "dev" {
		return CheckResult{Current: s.Current, Latest: latest}, nil
	}
	return CheckResult{
		Current:   s.Current,
		Latest:    latest,
		Available: updater.Compare(s.Current, latest) < 0,
	}, nil
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
	return replaceExecutable(dst)
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
