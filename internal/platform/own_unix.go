//go:build unix || linux || darwin

package platform

import (
	"os"
	"os/user"
	"strconv"
)

// ChownToSudoUser hands ownership of path to the user who invoked sudo.
// No-op when not root or SUDO_USER is unset. Best-effort: never fails hard.
func ChownToSudoUser(path string) {
	if os.Geteuid() != 0 || path == "" {
		return
	}
	name := os.Getenv("SUDO_USER")
	if name == "" {
		return
	}
	u, err := user.Lookup(name)
	if err != nil {
		return
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return
	}
	_ = os.Chown(path, uid, gid)
}
