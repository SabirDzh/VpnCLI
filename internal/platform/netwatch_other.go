//go:build !darwin

package platform

import (
	"fmt"
	"os"
	"strings"
)

// DefaultIface parses /proc/net/route for the default gateway route.
func DefaultIface() (string, error) {
	raw, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n")[1:] {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "00000000" && fields[7] == "00000000" {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no default route")
}
