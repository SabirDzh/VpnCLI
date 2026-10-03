//go:build windows

package process

import "os/exec"

func detach(cmd *exec.Cmd) {}
