package app

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestWatchMuxFallback(t *testing.T) {
	dir := t.TempDir()
	log := dir + "/sb.log"
	if err := os.WriteFile(log, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	hit := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go WatchMux(ctx, log, func() { hit <- struct{}{} })
	time.Sleep(100 * time.Millisecond)
	f, err := os.OpenFile(log, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("ERROR router: multiplex: handshake failed: connection refused\n")
	f.Close()
	select {
	case <-hit:
	case <-time.After(3 * time.Second):
		t.Fatal("fallback not triggered")
	}
}

func TestWatchMuxNoFalsePositive(t *testing.T) {
	dir := t.TempDir()
	log := dir + "/sb.log"
	if err := os.WriteFile(log, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	hit := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go WatchMux(ctx, log, func() { hit <- struct{}{} })
	time.Sleep(100 * time.Millisecond)
	f, err := os.OpenFile(log, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("INFO inbound/mixed[mixed-in]: connection from 127.0.0.1\n")
	f.Close()
	select {
	case <-hit:
		t.Fatal("benign line must not trigger fallback")
	case <-time.After(700 * time.Millisecond):
	}
}

func TestWatchMuxMissingFile(t *testing.T) {
	called := false
	done := make(chan struct{})
	go func() {
		WatchMux(context.Background(), "/nonexistent/dir/sb.log", func() { called = true })
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("must return immediately on missing file")
	}
	if called {
		t.Fatal("fallback must not fire")
	}
}
