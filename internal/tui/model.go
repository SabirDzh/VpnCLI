package tui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/tui/screen/profiles"
	"github.com/SabirDzh/VpnCLI/internal/tui/screen/status"
	"github.com/SabirDzh/VpnCLI/internal/tui/screen/subscriptions"
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

// Limits for the minimum usable window.
const (
	minWidth  = 60
	minHeight = 20
)

// Model is the root TUI model: tabs, routing, global keys.
type Model struct {
	deps    shared.Deps
	screens []shared.Screen
	active  int
	width   int
	height  int
	ready   bool
	styles  theme.Styles
	ctx     context.Context
	showAll bool // full help overlay (?)
}

// NewModel builds the root model with all screens.
func NewModel(ctx context.Context, deps shared.Deps) *Model {
	st := theme.Default()
	return &Model{
		deps:   deps,
		styles: st,
		ctx:    ctx,
		screens: []shared.Screen{
			status.New(ctx, deps.Connection, st, deps.ReadOnly),
			profiles.New(ctx, deps.Connection, deps.Profiles, st, deps.ReadOnly),
			subscriptions.New(deps, st),
		},
	}
}

// Init starts every screen's initial fetch.
func (m *Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.screens))
	for _, s := range m.screens {
		cmds = append(cmds, s.Init())
	}
	return tea.Batch(cmds...)
}

// Update routes global keys and forwards the rest to the active screen.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		return m, m.forward(msg)
	case tea.KeyPressMsg:
		if cmd, handled := m.globalKey(msg.String()); handled {
			return m, cmd
		}
		return m, m.forward(msg)
	default:
		return m, m.forward(msg)
	}
}

func (m *Model) forward(msg tea.Msg) tea.Cmd {
	next, cmd := m.screens[m.active].Update(msg)
	m.screens[m.active] = next
	return cmd
}

// globalKey handles tab switching, help and quit. Screens never see these.
func (m *Model) globalKey(k string) (tea.Cmd, bool) {
	switch k {
	case "q", "ctrl+c":
		return tea.Quit, true
	case "?":
		m.showAll = !m.showAll
		return nil, true
	case "1", "2", "3":
		n := int(k[0] - '1')
		return m.switchTo(n)
	case "tab":
		return m.switchTo((m.active + 1) % len(m.screens))
	case "shift+tab":
		return m.switchTo((m.active + len(m.screens) - 1) % len(m.screens))
	}
	return nil, false
}

func (m *Model) switchTo(n int) (tea.Cmd, bool) {
	if n < 0 || n >= len(m.screens) || n == m.active {
		return nil, n == m.active
	}
	m.active = n
	return m.screens[m.active].Init(), true
}

// View assembles header, body and footer into an alt-screen view.
func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m *Model) render() string {
	if !m.ready {
		return "loading…"
	}
	if m.width < minWidth || m.height < minHeight {
		return m.styles.Warn.Render(
			"Окно слишком маленькое. Увеличь до 60×20.") + "\n"
	}
	var b strings.Builder
	b.WriteString(m.header() + "\n")
	body := m.screens[m.active].View(m.width, m.height-12)
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	b.WriteString(m.footer())
	return b.String()
}

func (m *Model) header() string {
	var tabs []string
	for i, s := range m.screens {
		label := " " + tabKeyLabel(i) + " " + s.Title() + " "
		if i == m.active {
			tabs = append(tabs, m.styles.TabActive.Render(label))
		} else {
			tabs = append(tabs, m.styles.Tab.Render(label))
		}
	}
	banner := ""
	if m.deps.ReadOnly {
		banner = "  " + m.styles.Banner.Render("[read-only: без root]")
	}
	logo := m.styles.Logo.Render(strings.Join(logoLines, "\n"))
	logoRows := strings.Split(logo, "\n")
	tabLine := strings.Join(tabs, " ")
	var b strings.Builder
	b.WriteString(logoRows[0] + "   " + tabLine + "\n")
	b.WriteString(strings.Join(logoRows[1:], "\n") + banner)
	return b.String()
}

// logoLines is the ASCII logo rendered in the header.
var logoLines = []string{
	`__     __  _______   __    __`,
	`╱  │   ╱  │╱       ╲ ╱  ╲  ╱  │`,
	`$$ │   $$ │$$$$$$$  │$$  ╲ $$ │`,
	`$$ │   $$ │$$ │__$$ │$$$  ╲$$ │`,
	`$$  ╲ ╱$$╱ $$    $$╱ $$$$  $$ │`,
	` $$  ╱$$╱  $$$$$$$╱  $$ $$ $$ │`,
	`  $$ $$╱   $$ │      $$ │$$$$ │`,
	`   $$$╱    $$ │      $$ │ $$$ │`,
	`    $╱     $$╱       $$╱   $$╱`,
}

func tabKeyLabel(i int) string {
	labels := []string{"1", "2", "3"}
	if i < 0 || i >= len(labels) {
		return ""
	}
	return labels[i]
}

func (m *Model) footer() string {
	if m.showAll {
		return m.fullHelp()
	}
	var parts []string
	for _, b := range m.screens[m.active].Keys() {
		parts = append(parts, keyHint(b, m.styles))
	}
	parts = append(parts, m.styles.Key.Render("q")+" "+m.styles.Help.Render("quit (VPN stays on)"))
	parts = append(parts, m.styles.Key.Render("?")+" "+m.styles.Help.Render("help"))
	foot := strings.Join(parts, "  ")
	foot += "\n" + m.styles.Dim.Render("Выход из TUI не выключает VPN")
	return foot
}

func (m *Model) fullHelp() string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render("Клавиши") + "\n\n")
	rows := [][2]string{
		{"1 2 3, tab/shift+tab", "переключение экранов"},
		{"↑ ↓ / j k", "навигация"},
		{"enter", "основное действие экрана"},
		{"c", "подключить / отключить"},
		{"u / U", "обновить подписку / все подписки"},
		{"/", "фильтр (Profiles)"},
		{"r", "обновить данные"},
		{"?", "эта справка"},
		{"q, ctrl+c", "выход (VPN остаётся включённым)"},
	}
	for _, r := range rows {
		b.WriteString(m.styles.Help.Render(r[0]) + "  " + r[1] + "\n")
	}
	return b.String()
}
