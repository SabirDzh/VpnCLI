package tui

import "charm.land/bubbles/v2/key"

// Key bindings shared by the root model. Screens expose their own
// action bindings via Keys() for the footer.
var (
	keyQuit = key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit (VPN stays on)"),
	)
	keyHelp = key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	)
)
