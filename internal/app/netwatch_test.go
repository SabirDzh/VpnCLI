package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

var errFake = errors.New("fake poll error")

func TestWatchNetworkFiresOnChange(t *testing.T) {
	cur := "en0"
	poll := func() (string, error) { return cur, nil }
	changes := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go WatchNetwork(ctx, poll, 10*time.Millisecond, func() { changes <- struct{}{} })
	time.Sleep(30 * time.Millisecond)
	cur = "en11" // network change
	select {
	case <-changes:
	case <-time.After(1 * time.Second):
		t.Fatal("change not detected")
	}
	// no interface flip → no more events
	select {
	case <-changes:
		t.Fatal("duplicate event without a change")
	case <-time.After(60 * time.Millisecond):
	}
}

func TestWatchNetworkPollErrorsIgnored(t *testing.T) {
	fail := true
	poll := func() (string, error) {
		if fail {
			return "", errFake
		}
		return "en0", nil
	}
	changes := 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		WatchNetwork(ctx, poll, 5*time.Millisecond, func() { changes++ })
		close(done)
	}()
	time.Sleep(30 * time.Millisecond)
	fail = false
	// first successful poll just establishes the baseline, no event yet
	time.Sleep(30 * time.Millisecond)
	if changes != 0 {
		t.Fatalf("poll errors must not trigger changes, got %d", changes)
	}
}
