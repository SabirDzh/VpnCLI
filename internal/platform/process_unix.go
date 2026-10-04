//go:build unix || linux || darwin

package platform

import (
	"errors"
	"os"
	"syscall"
)

// Alive reports whether pid exists (signal 0 probe).
func Alive(pid int) bool {
	alive, _ := Signallable(pid)
	return alive
}

// Signallable reports whether pid exists and whether we may signal it.
// A root-owned daemon probed by a user returns (true, false).
func Signallable(pid int) (alive, permitted bool) {
	if pid <= 0 {
		return false, false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false, false
	}
	if err := p.Signal(syscall.Signal(0)); err == nil {
		return true, true
	} else {
		var errno syscall.Errno
		if errors.As(err, &errno) && errno == syscall.EPERM {
			return true, false // exists, no permission
		}
		return false, false
	}
}
