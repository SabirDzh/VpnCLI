package app

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/subscription"
)

type subStore interface {
	ListSubscriptions() ([]domain.Subscription, error)
	GetSubscription(idOrName string) (domain.Subscription, error)
	SaveSubscription(sub domain.Subscription) error
	DeleteSubscription(idOrName string) error
}

type subProfileStore interface {
	ListProfiles() ([]domain.Profile, error)
	SaveProfile(p domain.Profile) error
	DeleteProfile(id string) error
}

// SubscriptionService manages subscription sources and syncs profiles.
type SubscriptionService struct {
	subs     subStore
	profiles subProfileStore
	fetch    func(url string) ([]domain.Profile, error)
}

// NewSubscriptionService wires stores with the default HTTP fetcher.
func NewSubscriptionService(subs subStore, profiles subProfileStore) *SubscriptionService {
	return &SubscriptionService{subs: subs, profiles: profiles, fetch: subscription.Fetch}
}

// Add registers a subscription source.
func (s *SubscriptionService) Add(name, url string) (domain.Subscription, error) {
	h := sha256.Sum256([]byte(url))
	sub := domain.Subscription{ID: hex.EncodeToString(h[:])[:16], Name: name, URL: url}
	if err := s.subs.SaveSubscription(sub); err != nil {
		return domain.Subscription{}, err
	}
	return sub, nil
}

// List returns all subscriptions.
func (s *SubscriptionService) List() ([]domain.Subscription, error) {
	return s.subs.ListSubscriptions()
}

// Remove deletes a subscription (profiles stay; use Update/prune separately).
func (s *SubscriptionService) Remove(idOrName string) error {
	return s.subs.DeleteSubscription(idOrName)
}

// Edit changes a subscription's name and/or URL. The ID stays stable so
// profile source tags, counts and pruning keep working.
func (s *SubscriptionService) Edit(idOrName, name, url string) (domain.Subscription, error) {
	sub, err := s.subs.GetSubscription(idOrName)
	if err != nil {
		return domain.Subscription{}, err
	}
	if name != "" {
		sub.Name = name
	}
	if url != "" {
		sub.URL = url
	}
	if err := s.subs.SaveSubscription(sub); err != nil {
		return domain.Subscription{}, err
	}
	return sub, nil
}

// Update fetches the subscription and upserts profiles by stable ID.
// Profiles of this source missing from the remote are deleted.
func (s *SubscriptionService) Update(idOrName string) (int, error) {
	sub, err := s.subs.GetSubscription(idOrName)
	if err != nil {
		return 0, err
	}
	remote, err := s.fetch(sub.URL)
	if err != nil {
		return 0, err
	}
	src := domain.SubscriptionSource(sub.ID)
	want := make(map[string]bool, len(remote))
	for _, p := range remote {
		p.Source = src
		if p.Name == "" {
			p.Name = sub.Name + "/" + p.ID[:8]
		}
		if err := s.profiles.SaveProfile(p); err != nil {
			return 0, err
		}
		want[p.ID] = true
	}
	existing, err := s.profiles.ListProfiles()
	if err != nil {
		return 0, err
	}
	for _, p := range existing {
		if p.Source == src && !want[p.ID] {
			_ = s.profiles.DeleteProfile(p.ID)
		}
	}
	sub.LastUpdated = time.Now()
	_ = s.subs.SaveSubscription(sub)
	return len(remote), nil
}
