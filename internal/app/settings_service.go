package app

import (
	"sync"

	"github.com/SabirDzh/VpnCLI/internal/config"
)

// SettingsService applies TUI setting changes to the config file:
// load, mutate, validate, save atomically — under a lock.
type SettingsService struct {
	path string
	mu   sync.Mutex
}

// NewSettingsService creates the service bound to a config file path.
func NewSettingsService(path string) *SettingsService {
	return &SettingsService{path: path}
}

// Update loads the config, applies fn, validates and saves. The returned
// Config is the effective post-update state. A failed validation leaves
// the file untouched.
func (s *SettingsService) Update(fn func(c *config.Config)) (config.Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := config.Load(s.path, nil)
	if err != nil {
		return config.Config{}, err
	}
	fn(&c)
	if err := config.Save(s.path, c); err != nil {
		return config.Config{}, err
	}
	return c, nil
}
