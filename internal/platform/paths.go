// Package platform centralizes OS-dependent paths, privilege checks
// and process helpers. Behavior splits live in _unix/_windows files.
package platform

import (
	"os"
	"path/filepath"
)

// Paths — все файловые расположения приложения.
type Paths struct {
	ConfigDir  string
	DataDir    string
	StateFile  string
	RuntimeDir string
	LogFile    string
	// Privileged is true when paths point at system locations (/etc, /var).
	Privileged bool
}

// Resolve returns paths for the current user.
// Root (incl. sudo) gets system paths (/etc/vpn, /var/lib/vpn, /run/vpn),
// unprivileged users get XDG home paths. See paths_unix.go.
func Resolve() (Paths, error) {
	return resolve()
}

// Ensure creates directories with safe permissions.
func (p Paths) Ensure() error {
	for _, d := range []string{p.ConfigDir, p.DataDir, p.RuntimeDir, filepath.Dir(p.LogFile)} {
		if d == "" {
			continue
		}
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
		ChownToSudoUser(d)
	}
	ChownToSudoUser(p.LogFile)
	return nil
}
