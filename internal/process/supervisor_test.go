package process

import (
	"context"
	"testing"
	"time"

	"github.com/SabirDzh/VpnCLI/internal/platform"
)

// TestStopLiveProcess verifies the supervisor lifecycle on a real process:
// detached start, graceful SIGTERM stop, idempotent second stop.
func TestStopLiveProcess(t *testing.T) {
	dir := t.TempDir()
	pid, err := Start(context.Background(), "/bin/sleep", []string{"60"}, dir+"/test.log")
	if err != nil {
		t.Fatal(err)
	}
	if !platform.Alive(pid) {
		t.Fatal("process should be alive")
	}
	start := time.Now()
	if err := Stop(pid, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("stop took too long: %v", d)
	}
	if platform.Alive(pid) {
		t.Fatal("process should be dead")
	}
	if err := Stop(pid, time.Second); err != nil {
		t.Fatal(err)
	}
}

// TestStartSurvivesContextCancel pins the detached-daemon contract: the
// child outlives the request context (the TUI cancels its 30s op context
// right after Up returns; the core must keep running).
func TestStartSurvivesContextCancel(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	pid, err := Start(ctx, "/bin/sleep", []string{"60"}, dir+"/test.log")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Stop(pid, 5*time.Second) })
	cancel()
	time.Sleep(500 * time.Millisecond)
	if !platform.Alive(pid) {
		t.Fatalf("detached child %d must survive request-context cancel", pid)
	}
}
