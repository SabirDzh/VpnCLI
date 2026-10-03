package singbox

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Minimum supported sing-box version (TUN/DNS format fixed for 1.14.x).
const MinVersion = "1.14.0"

var versionRe = regexp.MustCompile(`(?im)^sing-box version (\S+)`)

// FindBinary resolves the binary: explicit path first, then PATH.
func FindBinary(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	p, err := exec.LookPath("sing-box")
	if err != nil {
		return "", fmt.Errorf("sing-box binary not found (set core.singbox.path): %w", err)
	}
	return p, nil
}

// BinaryVersion runs `sing-box version` and parses the version.
func BinaryVersion(bin string) (string, error) {
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		return "", fmt.Errorf("run %s version: %w", bin, err)
	}
	m := versionRe.FindStringSubmatch(string(out))
	if m == nil {
		// fallback: first token that looks like X.Y.Z
		for _, f := range strings.Fields(string(out)) {
			if looksLikeVersion(f) {
				return strings.TrimPrefix(f, "v"), nil
			}
		}
		return "", fmt.Errorf("cannot parse sing-box version output")
	}
	return strings.TrimPrefix(m[1], "v"), nil
}

// CheckMinVersion ensures ver >= min (numeric X.Y.Z comparison).
func CheckMinVersion(ver, min string) error {
	if compare(ver, min) < 0 {
		return fmt.Errorf("sing-box version %s is below minimum %s", ver, min)
	}
	return nil
}

func looksLikeVersion(s string) bool {
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		if _, err := strconv.Atoi(strings.TrimRight(p, "-alpha-beta")); err != nil {
			return false
		}
	}
	return true
}

func compare(a, b string) int {
	pa := strings.Split(strings.TrimPrefix(a, "v"), ".")
	pb := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < 3; i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(leadingDigits(pa[i]))
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(leadingDigits(pb[i]))
		}
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}

func leadingDigits(s string) string {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return s[:i]
}
