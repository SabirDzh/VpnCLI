package app

import (
	"context"
	"time"
)

// IfacePoller returns the current default-route interface name.
type IfacePoller func() (string, error)

// WatchNetwork polls the default interface and fires onChange (at most
// once per actual change) until ctx is done. Poll errors are ignored:
// a transiently unreadable routing table is not a network change.
func WatchNetwork(ctx context.Context, poll IfacePoller, every time.Duration, onChange func()) {
	t := time.NewTicker(every)
	defer t.Stop()
	cur, err := poll()
	if err != nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			next, err := poll()
			if err != nil {
				continue
			}
			if next != cur {
				cur = next
				onChange()
			}
		}
	}
}
