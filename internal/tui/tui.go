// Package tui implements the interactive terminal UI on Bubble Tea v2.
package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
)

// Run starts the TUI and blocks until quit. The terminal is restored on
// regular exit, on signal cancellation via ctx, and on model panics.
// Quitting never touches the VPN: the core is an external process.
func Run(ctx context.Context, deps shared.Deps) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tui panic: %v", r)
		}
	}()
	p := tea.NewProgram(NewModel(ctx, deps), tea.WithContext(ctx))
	_, err = p.Run()
	return err
}
