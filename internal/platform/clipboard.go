package platform

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ReadClipboard returns the clipboard text via the platform tool.
func ReadClipboard() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return execOut("pbpaste")
	case "windows":
		return execOut("powershell", "-NoProfile", "-Command", "Get-Clipboard")
	case "linux":
		if out, err := execOut("wl-paste", "--no-newline"); err == nil {
			return out, nil
		}
		return execOut("xclip", "-selection", "clipboard", "-o")
	default:
		return "", fmt.Errorf("clipboard is not supported on %s", runtime.GOOS)
	}
}

// WriteClipboard places text on the clipboard.
func WriteClipboard(text string) error {
	switch runtime.GOOS {
	case "darwin":
		return execIn("pbcopy", text)
	case "windows":
		return execIn("clip", text)
	case "linux":
		if err := execIn("wl-copy", text); err == nil {
			return nil
		}
		return execIn("xclip", "-selection", "clipboard", text)
	default:
		return fmt.Errorf("clipboard is not supported on %s", runtime.GOOS)
	}
}

func execOut(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func execIn(name string, input string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(input)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
