//go:build darwin

package platform

import (
	"fmt"
	"os/exec"
	"regexp"
)

var defaultIfaceRe = regexp.MustCompile(`interface:\s+(\S+)`)

// DefaultIface returns the name of the default-route interface.
func DefaultIface() (string, error) {
	out, err := exec.Command("route", "-n", "get", "default").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("route get default: %w", err)
	}
	m := defaultIfaceRe.FindSubmatch(out)
	if m == nil {
		return "", fmt.Errorf("no default interface in route output")
	}
	return string(m[1]), nil
}
