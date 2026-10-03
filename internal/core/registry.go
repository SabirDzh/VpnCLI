package core

import (
	"fmt"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// Registry maps core names to factories and selects a core for a profile.
type Registry struct {
	factories map[string]Factory
	settings  map[string]Settings
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{factories: map[string]Factory{}, settings: map[string]Settings{}}
}

// Register adds a core factory. Called explicitly from main.go (no init()).
func (r *Registry) Register(name string, f Factory, s Settings) {
	r.factories[name] = f
	r.settings[name] = s
}

// Get builds the named core or returns ErrCoreNotFound.
func (r *Registry) Get(name string) (Core, error) {
	f, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrCoreNotFound, name)
	}
	return f(r.settings[name])
}

// Select returns the preferred core for a profile: preferred first if it
// Supports() the profile, otherwise the first supporting core.
func (r *Registry) Select(p domain.Profile, preferred string) (Core, error) {
	if preferred != "" {
		if c, err := r.Get(preferred); err == nil && c.Supports(p) {
			return c, nil
		}
	}
	for name := range r.factories {
		c, err := r.Get(name)
		if err != nil {
			continue
		}
		if c.Supports(p) {
			return c, nil
		}
	}
	return nil, fmt.Errorf("%w: protocol %s", domain.ErrUnsupportedProtocol, p.Protocol)
}

// Names lists registered core names.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.factories))
	for n := range r.factories {
		out = append(out, n)
	}
	return out
}
