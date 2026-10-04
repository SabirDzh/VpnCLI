// Package subscriptions implements the Subscriptions tab: subscription
// list with profile counts, single/all refresh with per-row spinner,
// result toasts. Row errors never crash the UI.
package subscriptions

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/tui/component"
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

// Model is the Subscriptions tab state.
type Model struct {
	profiles shared.ProfileAPI
	subs     shared.SubscriptionAPI
	styles   theme.Styles

	list    []domain.Subscription
	counts  map[string]int
	cursor  int
	loaded  bool
	errText string
	busy    bool   // an update is in flight
	busyID  string // subscription id being updated, "" for all
	spin    spinner.Model
	toast   component.Toast
	width   int
}

// New creates the Subscriptions tab.
func New(deps shared.Deps, st theme.Styles) *Model {
	return &Model{
		profiles: deps.Profiles,
		subs:     deps.Subs,
		styles:   st,
		spin:     spinner.New(spinner.WithSpinner(spinner.Dot)),
		toast:    component.NewToast(st),
	}
}

// Title implements shared.Screen.
func (m *Model) Title() string { return "Subscriptions" }

// Keys implements shared.Screen.
func (m *Model) Keys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update selected")),
		key.NewBinding(key.WithKeys("U"), key.WithHelp("U", "update all")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

// Init implements shared.Screen.
func (m *Model) Init() tea.Cmd { return shared.FetchSubs(m.profiles, m.subs) }

// Update implements shared.Screen.
func (m *Model) Update(msg tea.Msg) (shared.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case shared.SubsMsg:
		m.loaded = true
		if msg.Err != nil {
			m.errText = shared.DescribeError(msg.Err)
			return m, nil
		}
		m.errText = ""
		m.list = msg.Subs
		m.counts = msg.Counts
		if m.cursor >= len(m.list) {
			m.cursor = max(0, len(m.list)-1)
		}
		return m, nil
	case shared.OpDoneMsg:
		m.busy = false
		m.busyID = ""
		cmds := []tea.Cmd{shared.FetchSubs(m.profiles, m.subs)}
		if msg.Err != nil {
			label := msg.Label
			if label == "" {
				label = "обновление"
			}
			cmds = append(cmds, m.showToast(label+": "+shared.DescribeError(msg.Err), false))
			return m, tea.Batch(cmds...)
		}
		cmds = append(cmds, m.showToast(fmt.Sprintf("Обновлено профилей: %d", msg.N), true))
		return m, tea.Batch(cmds...)
	case shared.ToastExpiredMsg:
		m.toast.Expire(msg.ID)
		return m, nil
	case spinner.TickMsg:
		if m.busy {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
		return m, nil
	case tea.KeyPressMsg:
		return m.onKey(msg.String())
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	}
	return m, nil
}

func (m *Model) onKey(k string) (shared.Screen, tea.Cmd) {
	switch k {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down", "j":
		if m.cursor < len(m.list)-1 {
			m.cursor++
		}
		return m, nil
	case "r":
		if m.busy {
			return m, nil
		}
		return m, shared.FetchSubs(m.profiles, m.subs)
	case "esc":
		return m, shared.Back()
	case "u", "U":
		if m.busy || !m.loaded || len(m.list) == 0 {
			return m, nil
		}
		m.busy = true
		if k == "u" {
			sub := m.list[m.cursor]
			m.busyID = sub.ID
			return m, tea.Batch(m.spin.Tick, shared.DoSubUpdate(m.subs, sub.ID, false))
		}
		m.busyID = ""
		return m, tea.Batch(m.spin.Tick, shared.DoSubUpdate(m.subs, "", true))
	}
	return m, nil
}

func (m *Model) showToast(text string, ok bool) tea.Cmd {
	return m.toast.Show(text, ok, func(id int) tea.Msg {
		return shared.ToastExpiredMsg{ID: id}
	})
}

// View implements shared.Screen.
func (m *Model) View(_, _ int) string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render(fmt.Sprintf("Subscriptions (%d)", len(m.list))) + "\n")
	b.WriteString(m.styles.Dim.Render("u — обновить · U — обновить все") + "\n\n")
	if !m.loaded {
		return m.styles.Dim.Render("loading…") + "\n"
	}
	if len(m.list) == 0 {
		b.WriteString(m.styles.Dim.Render("Нет подписок. Добавь через: vpn sub add <имя> <url>") + "\n")
	} else {
		b.WriteString(m.styles.Dim.Render(fmt.Sprintf("%-22s %10s  %s", "NAME", "PROFILES", "UPDATED")) + "\n")
	}
	for i, s := range m.list {
		cursor := "  "
		if i == m.cursor {
			cursor = m.styles.ActiveMark.Render("> ")
		}
		name := s.Name
		if m.busy && (m.busyID == "" || m.busyID == s.ID) {
			name += " " + m.spin.View()
		}
		if len(name) > 20 {
			name = name[:19] + "…"
		}
		updated := m.styles.Dim.Render("never")
		if !s.LastUpdated.IsZero() {
			updated = s.LastUpdated.Format("02.01 15:04")
		}
		row := fmt.Sprintf("%s%-20s %3d profiles  %s", cursor, name, m.counts[domain.SubscriptionSource(s.ID)], updated)
		if i == m.cursor {
			row = m.styles.SelectedRow.Render(row)
		}
		b.WriteString(row + "\n\n")
	}
	if m.errText != "" {
		b.WriteString(m.styles.Err.Render(m.errText) + "\n")
	}
	if t := m.toast.View(); t != "" {
		b.WriteString(t + "\n")
	}
	return shared.IndentLines(b.String(), " ")
}
