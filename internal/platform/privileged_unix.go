//go:build unix || linux || darwin

package platform

import "os"

// IsPrivileged reports whether the process can manage TUN (root).
func IsPrivileged() bool { return os.Geteuid() == 0 }
