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
