//go:build windows

package platform

import (
	"os"
	"path/filepath"
)

func resolve() (Paths, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, err
		}
		base = filepath.Join(home, "AppData", "Local")
	}
	cfgBase := os.Getenv("APPDATA")
	if cfgBase == "" {
		cfgBase = base
	}
	data := filepath.Join(base, "vpn")
	run := filepath.Join(data, "run")
	return Paths{
		ConfigDir:  filepath.Join(cfgBase, "vpn"),
		DataDir:    data,
		StateFile:  filepath.Join(run, "state.json"),
		RuntimeDir: run,
		LogFile:    filepath.Join(data, "sing-box.log"),
	}, nil
}
