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

	HeaderBar   lipgloss.Style
	SelectedRow lipgloss.Style
	ActiveMark  lipgloss.Style
	Key         lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style
}

// protocolColors maps protocols to badge colors.
var protocolColors = map[string]string{
	"vless":       "13", // magenta
	"vmess":       "14", // cyan
	"trojan":      "11", // yellow
	"shadowsocks": "10", // green
	"hysteria2":   "12", // blue
}

// Default returns the MVP palette.
func Default() Styles {
	s := Styles{
		Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		Tab:         lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(0, 1),
		TabActive:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("4")).Padding(0, 1),
		OK:          lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		Err:         lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		Warn:        lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		Dim:         lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		Box:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
		ToastOK:     lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true),
		ToastErr:    lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),
		Banner:      lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		HeaderBar:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("4")).Padding(0, 1),
		SelectedRow: lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("8")).Padding(0, 1),
		ActiveMark:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")),
		Key:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")),
		Label:       lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(10),
		Value:       lipgloss.NewStyle().Foreground(lipgloss.Color("15")),
	}
	return s
}

// ProtoBadge renders a colored [PROTOCOL] badge. Unknown protocols are dim.
func (s Styles) ProtoBadge(proto string) string {
	color, ok := protocolColors[proto]
	if !ok {
		return s.Dim.Render("[" + proto + "]")
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color)).Render("[" + proto + "]")
}
