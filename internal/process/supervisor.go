// Package process supervises external core binaries:
// detached start (setsid), log to file, SIGTERM with timeout then SIGKILL.
package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/SabirDzh/VpnCLI/internal/platform"
)

// Start launches bin with args detached from the terminal.
// stdout/stderr append to logFile. Returns the child pid.
func Start(ctx context.Context, bin string, args []string, logFile string) (int, error) {
	lf, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, fmt.Errorf("open log: %w", err)
	}
	platform.ChownToSudoUser(logFile)
	// fd intentionally left open in child; parent closes its copy.
	defer lf.Close()

	// The daemon outlives the request that started it (the TUI cancels its
	// op context right after Up returns), so spawn without cancellation.
	cmd := exec.CommandContext(context.WithoutCancel(ctx), bin, args...)
	cmd.Stdout = lf
	cmd.Stderr = lf
	cmd.Stdin = nil
	detach(cmd)

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start %s: %w", bin, err)
	}
	// Reap in background so zombies don't accumulate when CLI exits.
	go func() { _ = cmd.Wait() }()
	return cmd.Process.Pid, nil
}

// Stop terminates pid gracefully: SIGTERM, wait up to timeout, then SIGKILL.
func Stop(pid int, timeout time.Duration) error {
	if !platform.Alive(pid) {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	_ = p.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !platform.Alive(pid) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !platform.Alive(pid) {
		return nil
	}
	_ = p.Signal(syscall.SIGKILL)
	time.Sleep(300 * time.Millisecond)
	if platform.Alive(pid) {
		return fmt.Errorf("process %d did not exit", pid)
	}
	return nil
}
