// Package app holds business services. CLI commands are thin wrappers
// around these services.
package app

import (
	"fmt"

	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/subscription/uri"
)

type profileStore interface {
	ListProfiles() ([]domain.Profile, error)
	GetProfile(idOrName string) (domain.Profile, error)
	SaveProfile(p domain.Profile) error
	DeleteProfile(id string) error
	SetActiveProfile(idOrName string) (domain.Profile, error)
	ActiveProfile() (domain.Profile, error)
}

// ProfileService manages manual profiles.
type ProfileService struct {
	store profileStore
}

// NewProfileService creates the service.
func NewProfileService(s profileStore) *ProfileService { return &ProfileService{store: s} }

// AddFromURI parses uriStr and stores it as a manual profile.
func (s *ProfileService) AddFromURI(uriStr string) (domain.Profile, error) {
	p, err := uri.Parse(uriStr)
	if err != nil {
		return domain.Profile{}, err
	}
	p.Source = domain.ManualSource
	if err := s.store.SaveProfile(p); err != nil {
		return domain.Profile{}, err
	}
	return p, nil
}

// AddRaw imports a native core config blob as-is (builder passes it through).
func (s *ProfileService) AddRaw(name string, proto domain.Protocol, raw []byte) (domain.Profile, error) {
	if len(raw) == 0 {
		return domain.Profile{}, fmt.Errorf("%w: empty raw config", domain.ErrParse)
	}
	p := domain.Profile{
		Protocol: proto,
		Source:   domain.ManualSource,
		Name:     name,
		Raw:      raw,
	}
	p.ID = domain.ComputeID(proto, name, 0, string(raw)[:min(64, len(raw))])
	if err := s.store.SaveProfile(p); err != nil {
		return domain.Profile{}, err
	}
	return p, nil
}

// List returns all profiles.
func (s *ProfileService) List() ([]domain.Profile, error) { return s.store.ListProfiles() }

// Remove deletes a profile.
func (s *ProfileService) Remove(idOrName string) error { return s.store.DeleteProfile(idOrName) }

// Use selects the active profile.
func (s *ProfileService) Use(idOrName string) (domain.Profile, error) {
	return s.store.SetActiveProfile(idOrName)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
