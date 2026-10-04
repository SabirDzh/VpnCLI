package component

import (
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
// The cursor starts on Yes: opening the dialog already expresses intent.
func (c *Confirm) Ask(title, tag string) {
	c.title = title
	c.tag = tag
	c.show = true
	c.cursor = true
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

// View renders the modal box, left-aligned like the input modal.
func (c Confirm) View() string {
	if !c.show {
		return ""
	}
	yes, no := "[Yes]", "[ No ]"
	if !c.cursor {
		yes, no = c.styles.Dim.Render("[Yes]"), c.styles.SelectedRow.Render("[ No ]")
	} else {
		yes, no = c.styles.SelectedRow.Render("[Yes]"), c.styles.Dim.Render("[ No ]")
	}
	body := c.title + "\n\n" + no + "  " + yes + "\n\n" + c.styles.Dim.Render("←/→ или tab — выбор, enter — ок")
	return c.styles.Box.Render(body)
}
