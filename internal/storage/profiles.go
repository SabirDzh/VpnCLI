package storage

import (
	"encoding/json"
	"fmt"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

type profileFile struct {
	Profiles []domain.Profile `json:"profiles"`
	ActiveID string           `json:"activeId"`
}

// ListProfiles returns all stored profiles.
func (s *Store) ListProfiles() ([]domain.Profile, error) {
	pf, err := s.loadProfiles()
	if err != nil {
		return nil, err
	}
	return pf.Profiles, nil
}

// GetProfile returns a profile by id or name.
func (s *Store) GetProfile(idOrName string) (domain.Profile, error) {
	pf, err := s.loadProfiles()
	if err != nil {
		return domain.Profile{}, err
	}
	for _, p := range pf.Profiles {
		if p.ID == idOrName || p.Name == idOrName {
			return p, nil
		}
	}
	return domain.Profile{}, fmt.Errorf("%w: %s", domain.ErrProfileNotFound, idOrName)
}

// SaveProfile upserts a profile by ID.
func (s *Store) SaveProfile(p domain.Profile) error {
	return withLock(s.profilesPath(), func() error {
		pf, err := s.loadProfiles()
		if err != nil {
			return err
		}
		found := false
		for i, e := range pf.Profiles {
			if e.ID == p.ID {
				pf.Profiles[i] = p
				found = true
				break
			}
		}
		if !found {
			pf.Profiles = append(pf.Profiles, p)
		}
		return s.saveProfiles(pf)
	})
}

// DeleteProfile removes a profile; clears active selection if needed.
func (s *Store) DeleteProfile(id string) error {
	return withLock(s.profilesPath(), func() error {
		pf, err := s.loadProfiles()
		if err != nil {
			return err
		}
		out := pf.Profiles[:0]
		found := false
		for _, p := range pf.Profiles {
			if p.ID == id || p.Name == id {
				found = true
				continue
			}
			out = append(out, p)
		}
		if !found {
			return fmt.Errorf("%w: %s", domain.ErrProfileNotFound, id)
		}
		pf.Profiles = out
		if pf.ActiveID == id {
			pf.ActiveID = ""
		}
		// also clear if active was addressed by name
		for _, p := range pf.Profiles {
			if p.Name == pf.ActiveID {
				goto keep
			}
		}
	keep:
		return s.saveProfiles(pf)
	})
}

// SetActiveProfile selects the active profile by id or name.
func (s *Store) SetActiveProfile(idOrName string) (domain.Profile, error) {
	var selected domain.Profile
	err := withLock(s.profilesPath(), func() error {
		pf, err := s.loadProfiles()
		if err != nil {
			return err
		}
		for _, p := range pf.Profiles {
			if p.ID == idOrName || p.Name == idOrName {
				pf.ActiveID = p.ID
				selected = p
				return s.saveProfiles(pf)
			}
		}
		return fmt.Errorf("%w: %s", domain.ErrProfileNotFound, idOrName)
	})
	return selected, err
}

// ActiveProfile returns the selected profile, or ErrNoActiveProfile.
func (s *Store) ActiveProfile() (domain.Profile, error) {
	pf, err := s.loadProfiles()
	if err != nil {
		return domain.Profile{}, err
	}
	if pf.ActiveID == "" {
		return domain.Profile{}, domain.ErrNoActiveProfile
	}
	for _, p := range pf.Profiles {
		if p.ID == pf.ActiveID {
			return p, nil
		}
	}
	return domain.Profile{}, domain.ErrNoActiveProfile
}

func (s *Store) loadProfiles() (profileFile, error) {
	var pf profileFile
	data, err := readFile(s.profilesPath())
	if err != nil {
		return pf, err
	}
	if len(data) == 0 {
		return pf, nil
	}
	if err := json.Unmarshal(data, &pf); err != nil {
		return pf, fmt.Errorf("decode profiles: %w", err)
	}
	return pf, nil
}

func (s *Store) saveProfiles(pf profileFile) error {
	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(s.profilesPath(), data)
}
