// Package profiles implements the Profiles tab: plain cursor list styled
// like the main menu, activation (enter), activation with connect (c, with
// reconnect confirm), manual refresh (r), add/edit/delete (a, e, x) and
// substring filter (/).
// Secrets (UUID, passwords, hosts) never reach the list rows.
package profiles

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/tui/component"
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

// Model is the Profiles tab state.
type Model struct {
	conn     shared.ConnectionAPI
	profiles shared.ProfileAPI
	styles   theme.Styles
	readOnly bool
	ctx      context.Context

	items      []domain.Profile
	activeID   string
	activeName string
	loaded     bool
	errText    string
	busy       bool
	// connectAfterUse chains a connect behind the pending activation.
	connectAfterUse bool
	confirm         component.Confirm
	toast           component.Toast
	input           component.Input
	pending         domain.Profile // target awaiting confirm verdict or being edited
	// editStep drives the two-step edit form: 0 idle, 1 name, 2 uri.
	editStep  int
	editName  string
	cursor    int
	offset    int
	filter    string
	filtering bool
	width     int
	height    int
}

// New creates the Profiles tab.
func New(ctx context.Context, conn shared.ConnectionAPI, profiles shared.ProfileAPI, st theme.Styles, readOnly bool) *Model {
	return &Model{
		conn: conn, profiles: profiles, styles: st, readOnly: readOnly, ctx: ctx,
		toast: component.NewToast(st), confirm: component.NewConfirm(st),
		input: component.NewInput(st),
	}
}

// Title implements shared.Screen.
func (m *Model) Title() string { return "Profiles" }

// Keys implements shared.Screen.
func (m *Model) Keys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "use profile")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "use + connect")),
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add profile")),
		key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit profile")),
		key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete profile")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

// Init implements shared.Screen.
func (m *Model) Init() tea.Cmd { return shared.FetchProfiles(m.profiles) }

// filtered returns items matching the filter (name, protocol, source).
func (m *Model) filtered() []domain.Profile {
	if m.filter == "" {
		return m.items
	}
	q := strings.ToLower(m.filter)
	var out []domain.Profile
	for _, p := range m.items {
		hay := strings.ToLower(p.Name + " " + string(p.Protocol) + " " + p.Source)
		if strings.Contains(hay, q) {
			out = append(out, p)
		}
	}
	return out
}

func (m *Model) clamp() {
	n := len(m.filtered())
	if n == 0 {
		m.cursor, m.offset = 0, 0
		return
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	// keep cursor inside the visible window
	win := m.winHeight()
	if win < 1 {
		win = 1
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+win {
		m.offset = m.cursor - win + 1
	}
}

// winHeight is the row budget for the list.
func (m *Model) winHeight() int {
	h := m.height - 6 // title + hints + margins consumed by root chrome
	if h < 1 {
		h = 1
	}
	return h
}

// Update implements shared.Screen.
func (m *Model) Update(msg tea.Msg) (shared.Screen, tea.Cmd) {
	if m.confirm.Showing() {
		return m.updateConfirm(msg)
	}
	if m.editStep != 0 {
		return m.updateEdit(msg)
	}
	if m.input.Showing() {
		return m.updateInput(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clamp()
		return m, nil
	case shared.ProfilesMsg:
		m.loaded = true
		if msg.Err != nil {
			m.errText = shared.DescribeError(msg.Err)
			return m, nil
		}
		m.errText = ""
		m.activeID = msg.ActiveID
		m.activeName = ""
		m.items = msg.List
		for _, p := range msg.List {
			if p.ID == msg.ActiveID {
				m.activeName = p.Name
			}
		}
		m.clamp()
		return m, nil
	case shared.OpDoneMsg:
		if msg.Err != nil {
			m.busy = false
			m.connectAfterUse = false
			return m, m.showToast(shared.DescribeError(msg.Err), false)
		}
		switch msg.Op {
		case "use":
			if m.connectAfterUse {
				m.connectAfterUse = false
				return m, shared.DoUp(m.ctx, m.conn, m.pending.ID, m.pending.Name)
			}
			m.busy = false
			return m, shared.FetchProfiles(m.profiles)
		case "up":
			m.busy = false
			return m, tea.Batch(
				m.showToast("Подключено: "+msg.Label, true),
				shared.FetchProfiles(m.profiles),
			)
		case "add", "edit", "remove":
			m.busy = false
			var label string
			switch msg.Op {
			case "add":
				label = "Добавлен: "
			case "edit":
				label = "Изменён: "
			default:
				label = "Удалён: "
			}
			return m, tea.Batch(
				m.showToast(label+msg.Label, true),
				shared.FetchProfiles(m.profiles),
			)
		default:
			m.busy = false
			return m, nil
		}
	case shared.ToastExpiredMsg:
		m.toast.Expire(msg.ID)
		return m, nil
	case tea.KeyPressMsg:
		return m.onKey(msg.String())
	}
	return m, nil
}

func (m *Model) selected() (domain.Profile, bool) {
	items := m.filtered()
	if !m.loaded || m.cursor < 0 || m.cursor >= len(items) {
		return domain.Profile{}, false
	}
	return items[m.cursor], true
}

func (m *Model) onKey(k string) (shared.Screen, tea.Cmd) {
	if m.filtering {
		return m.filterKey(k)
	}
	switch k {
	case "up", "k":
		m.moveCursor(-1)
		return m, nil
	case "down", "j":
		m.moveCursor(1)
		return m, nil
	case "r":
		if m.busy {
			return m, nil
		}
		return m, shared.FetchProfiles(m.profiles)
	case "enter":
		it, ok := m.selected()
		if !ok || m.busy {
			return m, nil
		}
		m.busy = true
		return m, shared.DoUse(m.profiles, it.ID, it.Name)
	case "c":
		it, ok := m.selected()
		if !ok || m.busy {
			return m, nil
		}
		if m.readOnly {
			return m, m.showToast(shared.DescribeError(domain.ErrNotPrivileged), false)
		}
		p, err := m.findProfile(it.ID)
		if err != nil {
			return m, m.showToast(shared.DescribeError(err), false)
		}
		m.pending = p
		m.confirm.Ask(fmt.Sprintf("Подключить %s?", it.Name), "connect")
		return m, nil
	case "/":
		if m.busy {
			return m, nil
		}
		m.filtering = true
		return m, nil
	case "a":
		if m.busy {
			return m, nil
		}
		m.input.Open("URI профиля:", "")
		return m, nil
	case "e":
		it, ok := m.selected()
		if !ok || m.busy {
			return m, nil
		}
		if it.Source != domain.ManualSource {
			return m, m.showToast(shared.DescribeError(domain.ErrProfileManaged), false)
		}
		m.pending = it
		m.editStep = 1
		m.input.Open("Имя профиля:", it.Name)
		return m, nil
	case "x":
		it, ok := m.selected()
		if !ok || m.busy {
			return m, nil
		}
		m.pending = it
		m.confirm.Ask(fmt.Sprintf("Удалить %s?", it.Name), "delete")
		return m, nil
	case "esc":
		return m, shared.Back()
	}
	return m, nil
}

// filterKey handles keystrokes while the filter is open.
func (m *Model) filterKey(k string) (shared.Screen, tea.Cmd) {
	switch k {
	case "esc":
		m.filtering = false
		m.filter = ""
		m.clamp()
	case "enter":
		m.filtering = false
		m.clamp()
	case "backspace":
		r := []rune(m.filter)
		if len(r) > 0 {
			m.filter = string(r[:len(r)-1])
		}
		m.clamp()
	default:
		if len(k) == 1 {
			m.filter += k
			m.cursor = 0
			m.offset = 0
			m.clamp()
		}
	}
	return m, nil
}

// findProfile reloads the full profile (list rows carry no secrets).
func (m *Model) findProfile(id string) (domain.Profile, error) {
	list, err := m.profiles.List()
	if err != nil {
		return domain.Profile{}, err
	}
	for _, p := range list {
		if p.ID == id {
			return p, nil
		}
	}
	return domain.Profile{}, domain.ErrProfileNotFound
}

func (m *Model) updateConfirm(msg tea.Msg) (shared.Screen, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch kp.String() {
	case "left", "right", "tab", "shift+tab":
		m.confirm.Move()
		return m, nil
	case "enter":
		tag, ok := m.confirm.Resolve()
		if !ok {
			return m, nil
		}
		m.busy = true
		if tag == "delete" {
			return m, shared.DoRemove(m.profiles, m.pending.ID, m.pending.Name)
		}
		m.connectAfterUse = true
		return m, shared.DoUse(m.profiles, m.pending.ID, m.pending.Name)
	case "esc", "q":
		m.confirm.Resolve()
		return m, nil
	}
	return m, nil
}

func (m *Model) moveCursor(d int) {
	m.cursor += d
	m.clamp()
}

// updateInput routes keys to the add-profile field.
func (m *Model) updateInput(msg tea.Msg) (shared.Screen, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch kp.String() {
	case "enter":
		uri := m.input.Value()
		m.input.Close()
		if uri == "" {
			return m, nil
		}
		m.busy = true
		return m, shared.DoAdd(m.profiles, uri)
	case "esc":
		m.input.Close()
		return m, nil
	default:
		m.input.Key(kp.String())
		return m, nil
	}
}

// updateEdit drives the two-step edit form: name, then URI.
func (m *Model) updateEdit(msg tea.Msg) (shared.Screen, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch kp.String() {
	case "enter":
		if m.editStep == 1 {
			name := m.input.Value()
			if name == "" {
				return m, nil
			}
			m.editName = name
			m.editStep = 2
			m.input.Open("Новый URI (пусто — оставить):", "")
			return m, nil
		}
		uriStr := m.input.Value()
		m.editStep = 0
		m.input.Close()
		m.busy = true
		return m, shared.DoProfileEdit(m.profiles, m.pending.ID, m.editName, uriStr, m.pending.Name)
	case "esc":
		m.editStep = 0
		m.input.Close()
		return m, nil
	default:
		m.input.Key(kp.String())
		return m, nil
	}
}

func (m *Model) showToast(text string, ok bool) tea.Cmd {
	return m.toast.Show(text, ok, func(id int) tea.Msg {
		return shared.ToastExpiredMsg{ID: id}
	})
}

// View implements shared.Screen.
func (m *Model) View(width, height int) string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render("Profiles") + m.styles.Dim.Render(fmt.Sprintf(" (%d)", len(m.items))))
	if m.activeName != "" {
		b.WriteString("  " + m.styles.ActiveMark.Render("● "+m.activeName))
	}
	b.WriteString("\n\n")
	switch {
	case !m.loaded:
		b.WriteString(m.styles.Dim.Render("loading…") + "\n")
	case len(m.items) == 0:
		b.WriteString(m.styles.Dim.Render("Нет профилей. Добавь через: vpn profile add <uri>") + "\n")
	default:
		items := m.filtered()
		if len(items) == 0 {
			b.WriteString(m.styles.Dim.Render("Ничего не найдено — esc сбрасывает фильтр") + "\n")
		}
		end := m.offset + m.winHeight()
		if end > len(items) {
			end = len(items)
		}
		for i := m.offset; i < end; i++ {
			p := items[i]
			mark := "  "
			if p.ID == m.activeID {
				mark = m.styles.ActiveMark.Render("* ")
			}
			row := fmt.Sprintf("%s%d. %-18s %s  %s",
				mark, i+1, shared.Truncate(p.Name, 18),
				m.styles.ProtoBadge(protoUpper(p.Protocol)),
				m.styles.Dim.Render(shortSource(p.Source)),
			)
			if i == m.cursor {
				row = m.styles.SelectedRow.Render(row)
			}
			b.WriteString(row + "\n")
		}
	}
	if m.filtering || m.filter != "" {
		b.WriteString(m.styles.Dim.Render("filter: "+m.filter+"▌") + "\n")
	}
	if m.errText != "" {
		b.WriteString(m.styles.Err.Render(m.errText) + "\n")
	}
	if t := m.toast.View(); t != "" {
		b.WriteString(t + "\n")
	}
	if m.input.Showing() {
		b.WriteString("\n" + m.input.View())
	}
	_ = height
	out := shared.IndentLines(b.String(), " ")
	// the confirm box hugs the left edge, outside the page indent
	if m.confirm.Showing() {
		out += "\n" + m.confirm.View()
	}
	return out
}

func protoUpper(p domain.Protocol) string {
	return strings.ToUpper(string(p))
}

func shortSource(s string) string {
	if s == domain.ManualSource {
		return "manual"
	}
	if strings.HasPrefix(s, "subscription:") {
		return "sub:" + s[len("subscription:"):]
	}
	return s
}
