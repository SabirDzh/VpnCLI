// Package theme holds the TUI palette and lipgloss styles.
// NO_COLOR and color downgrades are handled by Bubble Tea/lipgloss itself.
package theme

import "charm.land/lipgloss/v2"

// Styles groups every style used by screens and components.
type Styles struct {
	Title     lipgloss.Style
	Tab       lipgloss.Style
	TabActive lipgloss.Style
	OK        lipgloss.Style
	Err       lipgloss.Style
	Warn      lipgloss.Style
	Dim       lipgloss.Style
	Help      lipgloss.Style
	Box       lipgloss.Style
	ToastOK   lipgloss.Style
	ToastErr  lipgloss.Style
	Banner    lipgloss.Style
}

// Default returns the MVP palette.
func Default() Styles {
	return Styles{
		Title:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		Tab:       lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(0, 1),
		TabActive: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("4")).Padding(0, 1),
		OK:        lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		Err:       lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		Warn:      lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		Dim:       lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		Help:      lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		Box:       lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
		ToastOK:   lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true),
		ToastErr:  lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),
		Banner:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
	}
}
