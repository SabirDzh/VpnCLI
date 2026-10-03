//go:build unix || linux || darwin

package platform

import (
	"os"
	"os/user"
	"path/filepath"
)

func resolve() (Paths, error) {
	if os.Geteuid() == 0 {
		// Under sudo, share the invoking user's store so that
		// `vpn profile add` (user) + `sudo vpn up` (root) see
		// the same profiles. Pure-root sessions keep system paths.
		if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
			if u, err := user.Lookup(sudoUser); err == nil {
				return homePaths(u.HomeDir), nil
			}
		}
		return Paths{
			ConfigDir:  "/etc/vpn",
			DataDir:    "/var/lib/vpn",
			StateFile:  "/run/vpn/state.json",
			RuntimeDir: "/run/vpn",
			LogFile:    "/var/log/vpn/sing-box.log",
			Privileged: true,
		}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	return homePaths(home), nil
}

func homePaths(home string) Paths {
	base := filepath.Join(home, ".config", "vpn")
	data := filepath.Join(home, ".local", "share", "vpn")
	run := filepath.Join(home, ".cache", "vpn", "run")
	return Paths{
		ConfigDir:  base,
		DataDir:    data,
		StateFile:  filepath.Join(run, "state.json"),
		RuntimeDir: run,
		LogFile:    filepath.Join(data, "sing-box.log"),
	}
}
