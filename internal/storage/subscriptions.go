package storage

import (
	"encoding/json"
	"fmt"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// ListSubscriptions returns all subscriptions.
func (s *Store) ListSubscriptions() ([]domain.Subscription, error) {
	data, err := readFile(s.subsPath())
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var out []domain.Subscription
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode subscriptions: %w", err)
	}
	return out, nil
}

// SaveSubscription upserts by ID.
func (s *Store) SaveSubscription(sub domain.Subscription) error {
	return withLock(s.subsPath(), func() error {
		list, err := s.ListSubscriptions()
		if err != nil {
			return err
		}
		found := false
		for i, e := range list {
			if e.ID == sub.ID {
				list[i] = sub
				found = true
				break
			}
		}
		if !found {
			list = append(list, sub)
		}
		data, err := json.MarshalIndent(list, "", "  ")
		if err != nil {
			return err
		}
		return writeAtomic(s.subsPath(), data)
	})
}

// DeleteSubscription removes a subscription by id or name.
func (s *Store) DeleteSubscription(idOrName string) error {
	return withLock(s.subsPath(), func() error {
		list, err := s.ListSubscriptions()
		if err != nil {
			return err
		}
		out := list[:0]
		found := false
		for _, x := range list {
			if x.ID == idOrName || x.Name == idOrName {
				found = true
				continue
			}
			out = append(out, x)
		}
		if !found {
			return fmt.Errorf("%w: %s", domain.ErrSubscriptionNotFound, idOrName)
		}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		return writeAtomic(s.subsPath(), data)
	})
}

// GetSubscription fetches one subscription by id or name.
func (s *Store) GetSubscription(idOrName string) (domain.Subscription, error) {
	list, err := s.ListSubscriptions()
	if err != nil {
		return domain.Subscription{}, err
	}
	for _, x := range list {
		if x.ID == idOrName || x.Name == idOrName {
			return x, nil
		}
	}
	return domain.Subscription{}, fmt.Errorf("%w: %s", domain.ErrSubscriptionNotFound, idOrName)
}
