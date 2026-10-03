//go:build unix || linux || darwin

package platform

import (
	"errors"
	"os"
	"syscall"
)

// Alive reports whether pid exists (signal 0 probe).
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := p.Signal(syscall.Signal(0)); err != nil {
		var errno syscall.Errno
		if errors.As(err, &errno) && errno == syscall.EPERM {
			return true // exists, no permission
		}
		return false
	}
	return true
}
