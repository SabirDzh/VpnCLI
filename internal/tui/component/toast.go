// Package component holds reusable TUI widgets: toast and confirm dialog.
package component

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

// Toast is a transient message with auto-hide. Errors linger longer.
// The expire constructor lets the owner turn the tick into its own
// message type (avoids an import cycle with the parent package).
type Toast struct {
	text   string
	isErr  bool
	id     int
	show   bool
	styles theme.Styles
}

// NewToast creates a hidden toast.
func NewToast(st theme.Styles) Toast { return Toast{styles: st} }

// Show displays a message; ok=false renders it as an error.
// expire converts the internal tick into an owner message carrying the id.
func (t *Toast) Show(text string, ok bool, expire func(id int) tea.Msg) tea.Cmd {
	t.id++
	t.text = text
	t.isErr = !ok
	t.show = true
	id := t.id
	d := 3 * time.Second
	if t.isErr {
		d = 6 * time.Second
	}
	return tea.Tick(d, func(time.Time) tea.Msg { return expire(id) })
}

// Expire hides the toast if the id matches (stale ticks are ignored).
func (t *Toast) Expire(id int) {
	if id == t.id {
		t.show = false
	}
}

// Visible reports whether the toast is on screen.
func (t Toast) Visible() bool { return t.show }

// View renders the toast or an empty string.
func (t Toast) View() string {
	if !t.show {
		return ""
	}
	if t.isErr {
		return t.styles.ToastErr.Render("! " + t.text)
	}
	return t.styles.ToastOK.Render("✓ " + t.text)
}
