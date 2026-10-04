// Package settings implements the Settings tab: effective configuration,
// core diagnostics and environment paths, plus interactive feature
// controls (blocklists, split tunneling) persisted via SettingsAPI.
package settings

import (
	"fmt"
	"strconv"
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
	kindBlocklist
	kindKillSwitch
	kindAppFirewall
	kindExclude
	kindInclude
	kindCycle  // cat: mode|stack|loglevel|dnsstrategy|mux
	kindToggle // cat: tun|autoroute|strictroute|preset
	kindAppExclude
	kindAppInclude
	kindDNS // dns servers list editor
	kindNum // cat: mtu|mixedport
)

// row is one settings line.
type row struct {
	section string // rendered once as a header
	label   string
	value   string
	ok      *bool // nil = neutral, else green/red dot
	kind    rowKind
	cat     string // category inside kind (blocklist kind, cycle id, …)
}

// cycles map each cycle row to its value list; activate moves to the next.
var cycles = map[string][]string{
	"mode":        {"exclude", "include", "off"},
	"stack":       {"system", "gvisor", "mixed"},
	"loglevel":    {"debug", "info", "warn", "error"},
	"dnsstrategy": {"prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"},
	"mux":         {"auto", "on", "off"},
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
	// editKind/editCat identify the row being edited via input.
	editKind rowKind
	editCat  string
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
		{label: "tun", value: tunVal, ok: tunOK, kind: kindToggle, cat: "tun"},
		{label: "mtu", value: fmt.Sprintf("%d", info.MTU), kind: kindNum, cat: "mtu"},
		{label: "stack", value: info.Stack, kind: kindCycle, cat: "stack"},
		{label: "auto route", value: onOff(info.AutoRoute), kind: kindToggle, cat: "autoroute"},
		{label: "strict route", value: onOff(info.StrictRoute), kind: kindToggle, cat: "strictroute"},
		{label: "mixed proxy", value: fmt.Sprintf("127.0.0.1:%d", info.MixedPort), kind: kindNum, cat: "mixedport"},
		{section: "DNS"},
		{label: "servers", value: splitValue(info.DNSServers), kind: kindDNS},
		{label: "strategy", value: info.DNSStrategy, kind: kindCycle, cat: "dnsstrategy"},
		{section: "Split"},
		{label: "split mode", value: modeVal(info.SplitMode), kind: kindCycle, cat: "mode"},
		{label: "apps exclude", value: splitValue(info.SplitExcludeApps), kind: kindAppExclude},
		{label: "apps include", value: splitValue(info.SplitIncludeApps), kind: kindAppInclude},
		{label: "preset рф", value: onOff(info.PresetApps), kind: kindToggle, cat: "preset"},
		{label: "split exclude", value: splitValue(info.SplitExclude), kind: kindExclude},
		{label: "split include", value: splitValue(info.SplitInclude), kind: kindInclude},
		{section: "Blocklists"},
		{label: "adblock", value: onOff(info.Adblock), kind: kindBlocklist, cat: "ads"},
		{label: "trackerblock", value: onOff(info.TrackerBlock), kind: kindBlocklist, cat: "trackers"},
		{label: "socialblock", value: onOff(info.SocialBlock), kind: kindBlocklist, cat: "social"},
		{section: "Firewall"},
		{label: "app firewall", value: splitValue(info.AppFirewall), kind: kindAppFirewall},
		{label: "kill switch", value: onOff(info.KillSwitch), kind: kindKillSwitch},
		{label: "multiplex", value: info.Multiplex, kind: kindCycle, cat: "mux"},
		{section: "App"},
		{label: "log level", value: info.LogLevel, kind: kindCycle, cat: "loglevel"},
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

// modeVal shows the split mode in Russian; cycling uses raw values.
func modeVal(m string) string {
	switch m {
	case "exclude":
		return "кроме"
	case "include":
		return "только"
	case "off":
		return "выкл"
	}
	return m
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

// Keys implements shared.Screen. esc/back is appended by the root
// footer, so it is not declared here.
func (m *Model) Keys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter", "space"), key.WithHelp("enter/space", "toggle or edit")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
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
	case kindBlocklist:
		var cur bool
		switch r.cat {
		case "ads":
			cur = m.info.Adblock
		case "trackers":
			cur = m.info.TrackerBlock
		case "social":
			cur = m.info.SocialBlock
		}
		err = m.api.SetBlocklist(r.cat, !cur)
	case kindKillSwitch:
		err = m.api.SetKillSwitch(!m.info.KillSwitch)
	case kindCycle:
		err = m.nextCycle(r.cat)
	case kindToggle:
		err = m.toggleBool(r.cat)
	case kindNum:
		var title, cur string
		if r.cat == "mtu" {
			title = "MTU:"
			cur = fmt.Sprintf("%d", m.info.MTU)
		} else {
			title = "Mixed порт:"
			cur = fmt.Sprintf("%d", m.info.MixedPort)
		}
		m.editKind, m.editCat = kindNum, r.cat
		m.input.Open(title, cur)
		return m, nil
	case kindDNS:
		m.editKind = kindDNS
		m.input.Open("DNS серверы (через запятую, до 3):", strings.Join(m.info.DNSServers, ", "))
		return m, nil
	case kindAppFirewall:
		m.editKind = kindAppFirewall
		m.input.Open("App firewall (имена процессов через запятую):", strings.Join(m.info.AppFirewall, ", "))
		return m, nil
	case kindAppExclude:
		m.editKind = kindAppExclude
		m.input.Open("Приложения «кроме» (процессы через запятую):", strings.Join(m.info.SplitExcludeApps, ", "))
		return m, nil
	case kindAppInclude:
		m.editKind = kindAppInclude
		m.input.Open("Приложения «только» (процессы через запятую):", strings.Join(m.info.SplitIncludeApps, ", "))
		return m, nil
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

// nextCycle moves the current value of a cycle row to the next choice.
func (m *Model) nextCycle(cat string) error {
	choices := cycles[cat]
	var cur string
	switch cat {
	case "mode":
		cur = m.info.SplitMode
	case "stack":
		cur = m.info.Stack
	case "loglevel":
		cur = m.info.LogLevel
	case "dnsstrategy":
		cur = m.info.DNSStrategy
	case "mux":
		cur = m.info.Multiplex
	}
	next := choices[0]
	for i, c := range choices {
		if c == cur {
			next = choices[(i+1)%len(choices)]
			break
		}
	}
	switch cat {
	case "mode":
		return m.api.SetSplitMode(next)
	case "stack":
		return m.api.SetStack(next)
	case "loglevel":
		return m.api.SetLogLevel(next)
	case "dnsstrategy":
		return m.api.SetDNSstrategy(next)
	case "mux":
		return m.api.SetMultiplex(next)
	}
	return nil
}

// toggleBool flips a boolean toggle row.
func (m *Model) toggleBool(cat string) error {
	switch cat {
	case "tun":
		return m.api.SetTUNEnabled(!m.info.TUNEnabled)
	case "autoroute":
		return m.api.SetAutoRoute(!m.info.AutoRoute)
	case "strictroute":
		return m.api.SetStrictRoute(!m.info.StrictRoute)
	case "preset":
		return m.api.SetPresetApps(!m.info.PresetApps)
	}
	return nil
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
		kind, cat := m.editKind, m.editCat
		var list []string
		for _, tok := range strings.Split(value, ",") {
			tok = strings.TrimSpace(tok)
			if tok != "" {
				list = append(list, tok)
			}
		}
		if kind == kindNum {
			// parse before closing: invalid numbers keep the editor open
			n, perr := strconv.Atoi(strings.Join(list, ""))
			if perr != nil {
				return m, m.showToast("Ошибка: нужно число", false)
			}
			m.input.Close()
			m.editKind, m.editCat = kindNone, ""
			var err error
			if cat == "mtu" {
				err = m.api.SetMTU(n)
			} else {
				err = m.api.SetMixedPort(n)
			}
			if err != nil {
				// range/validation errors (config.Validate) reopen nothing:
				// the editor closed, the toast explains the problem.
				return m, m.showToast("Ошибка: "+err.Error(), false)
			}
			m.info = m.api.Snapshot()
			m.rebuild()
			return m, m.showToast("Сохранено — применится при следующем подключении", true)
		}
		m.input.Close()
		m.editKind, m.editCat = kindNone, ""
		var err error
		switch kind {
		case kindDNS:
			err = m.api.SetDNSServers(list)
		case kindAppFirewall:
			err = m.api.SetAppFirewall(list)
		case kindAppExclude:
			err = m.api.SetSplitApps("exclude", list)
		case kindAppInclude:
			err = m.api.SetSplitApps("include", list)
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
		m.editKind, m.editCat = kindNone, ""
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
	maxRows := height - 4
	if maxRows < 1 {
		maxRows = 1
	}
	// the window follows the cursor so every row stays reachable
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+maxRows {
		m.offset = m.cursor - maxRows + 1
	}
	if m.offset >= len(m.rows) {
		m.offset = max(0, len(m.rows)-1)
	}
	visible := m.rows[m.offset:]
	if len(visible) > maxRows {
		visible = visible[:maxRows]
	}
	for i, r := range visible {
		idx := m.offset + i
		if r.section != "" {
			b.WriteString(m.styles.Section.Render("── "+r.section+" ──") + "\n")
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
		rowText := "  " + dot + m.styles.FieldLabel.Render(fmt.Sprintf("%-14s", r.label)) +
			" " + m.styles.Value.Render(shared.Truncate(r.value, width-22))
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
