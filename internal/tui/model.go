package tui

import (
	"context"
	"fmt"
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

// Menu entries. Order is stable: status, profiles, subscriptions.
const (
	menuStatus = iota
	menuProfiles
	menuSubs
	menuCount
)

// Model is the root TUI model: main menu plus drill-down pages.
// Tabs are gone: everything opens from one list, esc goes back.
type Model struct {
	deps   shared.Deps
	styles theme.Styles
	ctx    context.Context

	cursor int
	page   shared.Screen // nil on the main menu
	pages  []shared.Screen

	width  int
	height int
	ready  bool

	showAll bool // full help overlay (?)

	// menu summaries, refreshed on menu and on return from a page
	st       shared.StatusMsg
	profiles shared.ProfilesMsg
	subs     shared.SubsMsg
}

// NewModel builds the root model with all pages.
func NewModel(ctx context.Context, deps shared.Deps) *Model {
	st := theme.Default()
	m := &Model{deps: deps, styles: st, ctx: ctx}
	m.pages = []shared.Screen{
		status.New(ctx, deps.Connection, st, deps.ReadOnly),
		profiles.New(ctx, deps.Connection, deps.Profiles, st, deps.ReadOnly),
		subscriptions.New(deps, st),
	}
	return m
}

// Init fetches menu summaries.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		shared.FetchStatus(m.ctx, m.deps.Connection),
		shared.FetchProfiles(m.deps.Profiles),
		shared.FetchSubs(m.deps.Profiles, m.deps.Subs),
	)
}

// Update routes global keys and forwards the rest to the open page.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		if m.page != nil {
			return m, m.forwardToPage(msg)
		}
		return m, nil
	case tea.KeyPressMsg:
		if cmd, handled := m.globalKey(msg.String()); handled {
			return m, cmd
		}
		if m.page != nil {
			return m, m.forwardToPage(msg)
		}
		return m, m.menuKey(msg.String())
	case shared.BackMsg:
		return m.back()
	default:
		m.store(msg)
		if m.page != nil {
			return m, m.forwardToPage(msg)
		}
		return m, nil
	}
}

// store keeps the latest summaries for the menu.
func (m *Model) store(msg tea.Msg) {
	switch msg := msg.(type) {
	case shared.StatusMsg:
		m.st = msg
	case shared.ProfilesMsg:
		m.profiles = msg
	case shared.SubsMsg:
		m.subs = msg
	}
}

func (m *Model) forwardToPage(msg tea.Msg) tea.Cmd {
	next, cmd := m.page.Update(msg)
	m.page = next
	return cmd
}

// back returns to the menu and refreshes its summaries.
func (m *Model) back() (tea.Model, tea.Cmd) {
	m.page = nil
	return m, m.Init()
}

// globalKey handles quit and help. Pages never see these.
func (m *Model) globalKey(k string) (tea.Cmd, bool) {
	switch k {
	case "q", "ctrl+c":
		return tea.Quit, true
	case "?":
		m.showAll = !m.showAll
		return nil, true
	}
	return nil, false
}

// menuKey navigates the main list and opens pages.
func (m *Model) menuKey(k string) tea.Cmd {
	switch k {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < menuCount-1 {
			m.cursor++
		}
	case "enter", "space":
		m.page = m.pages[m.cursor]
		return m.page.Init()
	case "1", "2", "3":
		n := int(k[0] - '1')
		if n >= 0 && n < len(m.pages) {
			m.cursor = n
			m.page = m.pages[n]
			return m.page.Init()
		}
	}
	return nil
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
	b.WriteString(m.header())
	if m.page != nil {
		body := m.page.View(m.width, m.height-13)
		b.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			b.WriteString("\n")
		}
	} else {
		b.WriteString(m.menu())
	}
	b.WriteString("\n" + m.footer())
	return b.String()
}

// menu renders the main list with live summaries.
// Rows breathe: 2-space indent, content clamped to [36,68] cells so the
// highlight bar stays uniform on wide terminals.
func (m *Model) menu() string {
	w := m.width - 6
	if w < 36 {
		w = 36
	}
	if w > 68 {
		w = 68
	}
	rows := []struct {
		title   string
		summary string
	}{
		{"Status", m.statusSummary()},
		{"Profiles", m.profilesSummary()},
		{"Subscriptions", m.subsSummary()},
	}
	var b strings.Builder
	for i, r := range rows {
		inner := fmt.Sprintf("%d. %-11s %s", i+1, r.title, shared.Truncate(r.summary, w-21))
		inner += strings.Repeat(" ", max(0, w-2-len([]rune(inner))))
		if i == m.cursor {
			b.WriteString(m.styles.SelectedRow.Render(m.styles.ActiveMark.Render("> ")+inner) + "\n")
		} else {
			b.WriteString(m.styles.Dim.Render("  "+inner) + "\n")
		}
	}
	return b.String()
}

func (m *Model) statusSummary() string {
	if m.st.Err != nil {
		return "error"
	}
	if m.st.St.Running {
		name := m.st.St.ProfileName
		if name == "" {
			name = "connected"
		}
		return "● " + name
	}
	return "○ disconnected"
}

func (m *Model) profilesSummary() string {
	n := len(m.profiles.List)
	active := ""
	for _, p := range m.profiles.List {
		if p.ID == m.profiles.ActiveID {
			active = p.Name
		}
	}
	if active == "" {
		return fmt.Sprintf("%d profiles", n)
	}
	return fmt.Sprintf("%d profiles · active: %s", n, active)
}

func (m *Model) subsSummary() string {
	return fmt.Sprintf("%d subs", len(m.subs.Subs))
}

func (m *Model) header() string {
	logo := m.styles.Logo.Render(strings.Join(logoLines, "\n"))
	banner := ""
	if m.deps.ReadOnly {
		banner = m.styles.Banner.Render("  [read-only: без root]")
	}
	// air around the logo: blank line above and below
	return "\n" + logo + banner + "\n\n"
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

func (m *Model) footer() string {
	if m.showAll {
		return m.fullHelp()
	}
	var parts []string
	if m.page != nil {
		for _, b := range m.page.Keys() {
			parts = append(parts, keyHint(b, m.styles))
		}
		parts = append(parts, m.styles.Key.Render("esc")+" "+m.styles.Help.Render("back"))
	} else {
		parts = append(parts,
			m.styles.Key.Render("↑↓")+" "+m.styles.Help.Render("navigate"),
			m.styles.Key.Render("enter")+" "+m.styles.Help.Render("open"),
		)
	}
	parts = append(parts,
		m.styles.Key.Render("q")+" "+m.styles.Help.Render("quit"),
		m.styles.Key.Render("?")+" "+m.styles.Help.Render("help"),
	)
	foot := "  " + strings.Join(parts, " | ")
	return foot
}

func (m *Model) fullHelp() string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render("Клавиши") + "\n\n")
	rows := [][2]string{
		{"↑ ↓ / j k", "навигация по меню"},
		{"enter", "открыть раздел"},
		{"esc", "назад в меню"},
		{"enter / c (Status)", "подключить / отключить"},
		{"enter (Profiles)", "сделать активным"},
		{"c (Profiles)", "активировать и подключить"},
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
