package tui

import (
	"charm.land/bubbles/v2/key"

	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

// keyHint renders one footer hint: bold key plus dim description.
func keyHint(b key.Binding, st theme.Styles) string {
	h := b.Help()
	return st.Key.Render(h.Key) + " " + st.Help.Render(h.Desc)
}
