package component

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

// Confirm is a modal Yes/No dialog. The owner shows it with Ask and
// resolves the verdict from the returned message.
type Confirm struct {
	title  string
	tag    string
	show   bool
	cursor bool // false=No, true=Yes
	styles theme.Styles
}

// NewConfirm creates a hidden dialog.
func NewConfirm(st theme.Styles) Confirm { return Confirm{styles: st} }

// Ask shows the dialog. tag identifies the pending action for the owner.
func (c *Confirm) Ask(title, tag string) {
	c.title = title
	c.tag = tag
	c.show = true
	c.cursor = false
}

// Showing reports whether the dialog is on screen.
func (c Confirm) Showing() bool { return c.show }

// Tag returns the pending action identifier.
func (c Confirm) Tag() string { return c.tag }

// Move toggles the cursor.
func (c *Confirm) Move() { c.cursor = !c.cursor }

// Resolve hides the dialog and returns (tag, verdict from the cursor).
func (c *Confirm) Resolve() (string, bool) {
	c.show = false
	return c.tag, c.cursor
}

// View renders the modal box.
func (c Confirm) View(width int) string {
	if !c.show {
		return ""
	}
	yes, no := "[ No ]", "[Yes]"
	if c.cursor {
		yes, no = "[Yes]", "[ No ]"
	}
	body := c.title + "\n\n" + no + "  " + yes + "\n\n" + c.styles.Dim.Render("←/→ или tab — выбор, enter — ок")
	box := c.styles.Box.Render(body)
	// center horizontally
	lines := strings.Split(box, "\n")
	w := 0
	for _, l := range lines {
		if lw := lipgloss.Width(l); lw > w {
			w = lw
		}
	}
	pad := (width - w) / 2
	if pad < 0 {
		pad = 0
	}
	for i, l := range lines {
		lines[i] = strings.Repeat(" ", pad) + l
	}
	return strings.Join(lines, "\n")
}
