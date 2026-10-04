// Package profiles implements the Profiles tab: filterable profile list,
// activation (enter), activation with connect (c, with reconnect confirm),
// manual refresh (r). Secrets never reach the list rows.
package profiles

import (
	"context"
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
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

	list     list.Model
	activeID string
	loaded   bool
	errText  string
	busy     bool
	// connectAfterUse chains a connect behind the pending activation.
	connectAfterUse bool
	confirm         component.Confirm
	toast           component.Toast
	pending         domain.Profile // connect target awaiting confirm verdict
	width           int
}

// profileItem is a list row without secrets (no UUID, password, host).
type profileItem struct {
	id       string
	name     string
	protocol string
	source   string
	active   bool
}

func (i profileItem) FilterValue() string { return i.name + " " + i.protocol }

type profileDelegate struct{ styles theme.Styles }

func (d profileDelegate) Height() int                         { return 1 }
func (d profileDelegate) Spacing() int                        { return 0 }
func (d profileDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (d profileDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	p, ok := item.(profileItem)
	if !ok {
		return
	}
	mark := "  "
	if p.active {
		mark = d.styles.ActiveMark.Render("* ")
	}
	row := fmt.Sprintf("%s%-20s %s  %s",
		mark, p.name,
		d.styles.ProtoBadge(strings.ToUpper(p.protocol)),
		d.styles.Dim.Render(shortSource(p.source)),
	)
	if index == m.Index() {
		row = d.styles.SelectedRow.Render(row)
	}
	fmt.Fprint(w, row)
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

// New creates the Profiles tab.
func New(ctx context.Context, conn shared.ConnectionAPI, profiles shared.ProfileAPI, st theme.Styles, readOnly bool) *Model {
	l := list.New(nil, profileDelegate{styles: st}, 0, 0)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	return &Model{
		conn: conn, profiles: profiles, styles: st, readOnly: readOnly, ctx: ctx,
		list: l, toast: component.NewToast(st), confirm: component.NewConfirm(st),
	}
}

// Title implements shared.Screen.
func (m *Model) Title() string { return "Profiles" }

// Keys implements shared.Screen.
func (m *Model) Keys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "use profile")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "use + connect")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

// Init implements shared.Screen.
func (m *Model) Init() tea.Cmd { return shared.FetchProfiles(m.profiles) }

// Update implements shared.Screen.
func (m *Model) Update(msg tea.Msg) (shared.Screen, tea.Cmd) {
	if m.confirm.Showing() {
		return m.updateConfirm(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.list.SetSize(msg.Width, msg.Height-2)
		return m, nil
	case shared.ProfilesMsg:
		m.loaded = true
		if msg.Err != nil {
			m.errText = shared.DescribeError(msg.Err)
			return m, nil
		}
		m.errText = ""
		m.activeID = msg.ActiveID
		items := make([]list.Item, 0, len(msg.List))
		for _, p := range msg.List {
			items = append(items, profileItem{
				id: p.ID, name: p.Name, protocol: string(p.Protocol),
				source: p.Source, active: p.ID == msg.ActiveID,
			})
		}
		return m, m.list.SetItems(items)
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
			return m, tea.Batch(
				m.showToast("Активен: "+msg.Label, true),
				shared.FetchProfiles(m.profiles),
			)
		case "up":
			m.busy = false
			return m, tea.Batch(
				m.showToast("Подключено: "+msg.Label, true),
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
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) selected() (profileItem, bool) {
	it, ok := m.list.SelectedItem().(profileItem)
	return it, ok && m.loaded
}

func (m *Model) onKey(k string) (shared.Screen, tea.Cmd) {
	switch k {
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
		return m, shared.DoUse(m.profiles, it.id, it.name)
	case "c":
		it, ok := m.selected()
		if !ok || m.busy {
			return m, nil
		}
		if m.readOnly {
			return m, m.showToast(shared.DescribeError(domain.ErrNotPrivileged), false)
		}
		p, err := m.findProfile(it.id)
		if err != nil {
			return m, m.showToast(shared.DescribeError(err), false)
		}
		m.pending = p
		m.confirm.Ask(fmt.Sprintf("Подключить %s?", it.name), "connect")
		return m, nil
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
		_ = tag
		if !ok {
			return m, nil
		}
		m.busy = true
		m.connectAfterUse = true
		return m, shared.DoUse(m.profiles, m.pending.ID, m.pending.Name)
	case "esc", "q":
		m.confirm.Resolve()
		return m, nil
	}
	return m, nil
}

func (m *Model) showToast(text string, ok bool) tea.Cmd {
	return m.toast.Show(text, ok, func(id int) tea.Msg {
		return shared.ToastExpiredMsg{ID: id}
	})
}

// View implements shared.Screen.
func (m *Model) View(width, height int) string {
	var b strings.Builder
	if !m.loaded {
		b.WriteString(m.styles.Dim.Render("loading…") + "\n")
	} else {
		b.WriteString(m.list.View() + "\n")
	}
	if m.errText != "" {
		b.WriteString(m.styles.Err.Render(m.errText) + "\n")
	}
	if t := m.toast.View(); t != "" {
		b.WriteString(t + "\n")
	}
	if m.confirm.Showing() {
		b.WriteString("\n" + m.confirm.View(width))
	}
	return b.String()
}
