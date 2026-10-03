// Package storage persists profiles, subscriptions and runtime state
// as JSON files with atomic writes (tmp + rename) and lock files.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SabirDzh/VpnCLI/internal/platform"
)

// Store is a file-backed repository root.
type Store struct {
	Dir       string // data dir: profiles.json, subscriptions.json
	StatePath string // runtime state.json
}

// New creates a Store, ensuring directories exist with 0700.
func New(dir, statePath string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if d := filepath.Dir(statePath); d != "" {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return nil, err
		}
	}
	return &Store{Dir: dir, StatePath: statePath}, nil
}

func (s *Store) profilesPath() string { return filepath.Join(s.Dir, "profiles.json") }
func (s *Store) subsPath() string     { return filepath.Join(s.Dir, "subscriptions.json") }

// withLock serializes writers via a .lock sidecar (poll + timeout).
func withLock(path string, fn func() error) error {
	lock := path + ".lock"
	deadline := time.Now().Add(5 * time.Second)
	for {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = f.Close()
			defer os.Remove(lock)
			return fn()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("storage locked: %s", path)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// writeAtomic writes data via tmp + rename with 0600.
func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	platform.ChownToSudoUser(tmp)
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	platform.ChownToSudoUser(path)
	return nil
}

func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}
