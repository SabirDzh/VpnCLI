// Package settings implements the Settings tab: effective configuration,
// core diagnostics and environment paths, plus interactive feature
// controls (blocklists, split tunneling) persisted via SettingsAPI.
package settings

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/tui/component"
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

// rowKind marks interactive rows.
type rowKind int

const (
	kindNone rowKind = iota
	kindAdblock
	kindTracker
	kindExclude
	kindInclude
)

// row is one settings line.
type row struct {
	section string // rendered once as a header
	label   string
	value   string
	ok      *bool // nil = neutral, else green/red dot
	kind    rowKind
}

// Model is the Settings tab state.
type Model struct {
	api    shared.SettingsAPI
	info   shared.SettingsInfo
	styles theme.Styles
	rows   []row
	cursor int
	offset int
	width  int
	toast  component.Toast
	input  component.Input
	// editKind is the row being edited via input; 0 = idle.
	editKind rowKind
}

// New creates the Settings tab. api may be nil: the page then renders
// read-only (no persisted writes).
func New(api shared.SettingsAPI, info shared.SettingsInfo, st theme.Styles) *Model {
	m := &Model{api: api, info: info, styles: st,
		toast: component.NewToast(st), input: component.NewInput(st)}
	m.rebuild()
	return m
}

func (m *Model) rebuild() {
	// keep the cursor on the same label across rebuilds
	label := ""
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		label = m.rows[m.cursor].label
	}
	info := m.info
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
	m.rows = []row{
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
		{section: "Features"},
		{label: "adblock", value: onOff(info.Adblock), kind: kindAdblock},
		{label: "trackerblock", value: onOff(info.TrackerBlock), kind: kindTracker},
		{label: "split exclude", value: splitValue(info.SplitExclude), kind: kindExclude},
		{label: "split include", value: splitValue(info.SplitInclude), kind: kindInclude},
		{section: "App"},
		{label: "log level", value: info.LogLevel},
		{label: "privileged", value: privVal, ok: privOK},
		{label: "config dir", value: info.ConfigDir},
		{label: "data dir", value: info.DataDir},
		{label: "state file", value: info.StateFile},
		{label: "log file", value: info.LogFile},
	}
	if label != "" {
		for i, r := range m.rows {
			if r.label == label && r.section == "" {
				m.cursor = i
				m.clamp()
				return
			}
		}
	}
	m.cursor = 0
	if len(m.rows) > 0 && m.rows[0].section != "" {
		m.skipSection(1)
	}
	m.clamp()
}

// skipSection moves the cursor by d rows, gliding over section headers.
func (m *Model) skipSection(d int) {
	for {
		next := m.cursor + d
		if next < 0 || next >= len(m.rows) {
			return
		}
		m.cursor = next
		if m.rows[m.cursor].section == "" {
			return
		}
	}
}

// splitValue renders a split list: count plus a preview of its entries.
func splitValue(list []string) string {
	if len(list) == 0 {
		return "empty"
	}
	return fmt.Sprintf("%d: %s", len(list), strings.Join(list, ", "))
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func (m *Model) clamp() {
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// Title implements shared.Screen.
func (m *Model) Title() string { return "Settings" }

// Keys implements shared.Screen.
func (m *Model) Keys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter", "space"), key.WithHelp("enter/space", "toggle or edit")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

// Init implements shared.Screen.
func (m *Model) Init() tea.Cmd { return nil }

// Update implements shared.Screen: navigation, toggles, list editing.
func (m *Model) Update(msg tea.Msg) (shared.Screen, tea.Cmd) {
	if m.input.Showing() {
		return m.updateInput(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case shared.ToastExpiredMsg:
		m.toast.Expire(msg.ID)
		return m, nil
	case tea.KeyPressMsg:
		return m.onKey(msg.String())
	}
	return m, nil
}

func (m *Model) onKey(k string) (shared.Screen, tea.Cmd) {
	switch k {
	case "up", "k":
		m.skipSection(-1)
		return m, nil
	case "down", "j":
		m.skipSection(1)
		return m, nil
	case "r":
		return m.refresh()
	case "enter", " ", "space":
		return m.activate()
	case "esc":
		return m, shared.Back()
	}
	return m, nil
}

func (m *Model) refresh() (shared.Screen, tea.Cmd) {
	if m.api != nil {
		m.info = m.api.Snapshot()
		m.rebuild()
	}
	return m, nil
}

// activate runs the action of the current row: toggle a feature or open
// the editor for a split list.
func (m *Model) activate() (shared.Screen, tea.Cmd) {
	if m.api == nil || m.cursor >= len(m.rows) {
		return m, nil
	}
	r := m.rows[m.cursor]
	if m.api == nil {
		return m, nil
	}
	var err error
	switch r.kind {
	case kindAdblock:
		err = m.api.SetAdblock(!m.info.Adblock)
	case kindTracker:
		err = m.api.SetTrackerBlock(!m.info.TrackerBlock)
	case kindExclude:
		m.editKind = kindExclude
		m.input.Open("Split exclude (домены или CIDR через запятую):", strings.Join(m.info.SplitExclude, ", "))
		return m, nil
	case kindInclude:
		m.editKind = kindInclude
		m.input.Open("Split include (домены или CIDR через запятую):", strings.Join(m.info.SplitInclude, ", "))
		return m, nil
	default:
		return m, nil
	}
	if err != nil {
		return m, m.showToast("Ошибка: "+err.Error(), false)
	}
	m.info = m.api.Snapshot()
	m.rebuild()
	return m, m.showToast("Сохранено — применится при следующем подключении", true)
}

func (m *Model) showToast(text string, ok bool) tea.Cmd {
	return m.toast.Show(text, ok, func(id int) tea.Msg {
		return shared.ToastExpiredMsg{ID: id}
	})
}

// updateInput drives the split-list editor: comma-separated tokens.
func (m *Model) updateInput(msg tea.Msg) (shared.Screen, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch kp.String() {
	case "enter":
		value := m.input.Value()
		m.input.Close()
		kind := m.editKind
		m.editKind = kindNone
		var list []string
		for _, tok := range strings.Split(value, ",") {
			tok = strings.TrimSpace(tok)
			if tok != "" {
				list = append(list, tok)
			}
		}
		var err error
		switch kind {
		case kindExclude:
			err = m.api.SetSplit(list, m.info.SplitInclude)
		case kindInclude:
			err = m.api.SetSplit(m.info.SplitExclude, list)
		}
		if err != nil {
			return m, m.showToast("Ошибка: "+err.Error(), false)
		}
		m.info = m.api.Snapshot()
		m.rebuild()
		return m, m.showToast("Сохранено — применится при следующем подключении", true)
	case "esc":
		m.input.Close()
		m.editKind = kindNone
		return m, nil
	default:
		m.input.Key(kp.String())
		return m, nil
	}
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
	for i, r := range visible {
		idx := m.offset + i
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
		rowText := fmt.Sprintf("  %s%-14s %s", dot, r.label,
			m.styles.Value.Render(shared.Truncate(r.value, width-22)))
		if idx == m.cursor && r.kind != kindNone {
			rowText = m.styles.SelectedRow.Render(strings.TrimLeft(rowText, " "))
		}
		b.WriteString(rowText + "\n")
	}
	if t := m.toast.View(); t != "" {
		b.WriteString("\n" + t + "\n")
	}
	if m.input.Showing() {
		b.WriteString("\n" + m.input.View() + "\n")
	}
	_ = width
	return shared.IndentLines(b.String(), " ")
}
