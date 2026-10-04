// Package settings implements the Settings tab: effective configuration,
// core diagnostics and environment paths. Read-only in MVP; values come
// from the composition root snapshot.
package settings

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

// row is one settings line.
type row struct {
	section string // rendered once as a header
	label   string
	value   string
	ok      *bool // nil = neutral, else green/red dot
}

// Model is the Settings tab state.
type Model struct {
	styles theme.Styles
	rows   []row
	offset int
	width  int
}

// New creates the Settings tab from a config snapshot.
func New(info shared.SettingsInfo, st theme.Styles) *Model {
	yes, no := true, false
	coreVal := info.SingBoxVersion
	coreOK := &yes
	if info.SingBoxErr != "" {
		coreVal = info.SingBoxErr
		coreOK = &no
	}
	tunVal := "off"
	tunOK := &no
	if info.TUNEnabled {
		tunVal = fmt.Sprintf("on · mtu %d", info.MTU)
		tunOK = &yes
	}
	privVal := "yes (root)"
	privOK := &yes
	if !info.Privileged {
		privVal = "no — TUN needs sudo"
		privOK = &no
	}
	path := info.SingBoxPath
	if path == "" {
		path = "PATH lookup"
	}
	return &Model{styles: st, rows: []row{
		{section: "Core"},
		{label: "engine", value: info.CoreDefault},
		{label: "sing-box", value: coreVal, ok: coreOK},
		{label: "min version", value: info.MinVersion},
		{label: "binary", value: path},
		{section: "Network"},
		{label: "tun", value: tunVal, ok: tunOK},
		{label: "auto route", value: onOff(info.AutoRoute)},
		{label: "strict route", value: onOff(info.StrictRoute)},
		{label: "mixed proxy", value: fmt.Sprintf("127.0.0.1:%d", info.MixedPort)},
		{section: "App"},
		{label: "log level", value: info.LogLevel},
		{label: "privileged", value: privVal, ok: privOK},
		{label: "config dir", value: info.ConfigDir},
		{label: "data dir", value: info.DataDir},
		{label: "state file", value: info.StateFile},
		{label: "log file", value: info.LogFile},
	}}
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// Title implements shared.Screen.
func (m *Model) Title() string { return "Settings" }

// Keys implements shared.Screen.
func (m *Model) Keys() []key.Binding { return nil }

// Init implements shared.Screen.
func (m *Model) Init() tea.Cmd { return nil }

// Update implements shared.Screen: vertical scroll for small windows.
func (m *Model) Update(msg tea.Msg) (shared.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.offset > 0 {
				m.offset--
			}
		case "down", "j":
			if m.offset < len(m.rows)-1 {
				m.offset++
			}
		case "esc":
			return m, shared.Back()
		}
	}
	return m, nil
}

// View implements shared.Screen.
func (m *Model) View(width, height int) string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render("Settings") + "\n\n")
	if m.offset >= len(m.rows) {
		m.offset = max(0, len(m.rows)-1)
	}
	visible := m.rows[m.offset:]
	maxRows := height - 4
	if maxRows < 1 {
		maxRows = 1
	}
	if len(visible) > maxRows {
		visible = visible[:maxRows]
	}
	for _, r := range visible {
		if r.section != "" {
			b.WriteString(m.styles.Dim.Render("── "+r.section+" ──") + "\n")
			continue
		}
		dot := ""
		if r.ok != nil {
			if *r.ok {
				dot = m.styles.OK.Render("● ")
			} else {
				dot = m.styles.Err.Render("● ")
			}
		}
		fmt.Fprintf(&b, "  %s%-14s %s\n",
			dot, r.label, m.styles.Value.Render(shared.Truncate(r.value, width-22)))
	}
	_ = width
	return shared.IndentLines(b.String(), " ")
}
