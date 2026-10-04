// Package subscriptions implements the Subscriptions tab: subscription
// list with profile counts, add/edit/delete (a, e, x), single/all refresh
// with per-row spinner, result toasts. Row errors never crash the UI.
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
	errs    map[string]string // last update error per subscription id
	cursor  int
	loaded  bool
	errText string
	busy    bool   // an update is in flight
	busyID  string // subscription id being updated, "" for all
	spin    spinner.Model
	toast   component.Toast
	input   component.Input
	confirm component.Confirm
	// inputStep drives the two-step add: 0 idle, 1 name, 2 url.
	inputStep int
	tmpName   string
	// editStep drives the two-step edit form: 0 idle, 1 name, 2 url.
	editStep int
	editName string
	pending  domain.Subscription
	width    int
}

// New creates the Subscriptions tab.
func New(deps shared.Deps, st theme.Styles) *Model {
	return &Model{
		profiles: deps.Profiles,
		subs:     deps.Subs,
		styles:   st,
		errs:     map[string]string{},
		spin:     spinner.New(spinner.WithSpinner(spinner.Dot)),
		toast:    component.NewToast(st),
		input:    component.NewInput(st),
		confirm:  component.NewConfirm(st),
	}
}

// Title implements shared.Screen.
func (m *Model) Title() string { return "Subscriptions" }

// Keys implements shared.Screen.
func (m *Model) Keys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update selected")),
		key.NewBinding(key.WithKeys("U"), key.WithHelp("U", "update all")),
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add subscription")),
		key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit subscription")),
		key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete subscription")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

// Init implements shared.Screen.
func (m *Model) Init() tea.Cmd { return shared.FetchSubs(m.profiles, m.subs) }

// Update implements shared.Screen.
func (m *Model) Update(msg tea.Msg) (shared.Screen, tea.Cmd) {
	if m.confirm.Showing() {
		return m.updateConfirm(msg)
	}
	if m.editStep != 0 {
		return m.updateEdit(msg)
	}
	if m.inputStep != 0 {
		return m.updateInput(msg)
	}
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
			if msg.Label != "" {
				m.errs[msg.Label] = shared.DescribeError(msg.Err)
			}
			cmds = append(cmds, m.showToast(label+": "+shared.DescribeError(msg.Err), false))
			return m, tea.Batch(cmds...)
		}
		if msg.Op == "update" && msg.Label != "" {
			delete(m.errs, msg.Label)
		}
		if msg.Op == "update-all" {
			m.errs = map[string]string{}
		}
		if msg.Op == "sub-add" {
			return m, tea.Batch(
				m.showToast("Добавлена: "+msg.Label, true),
				shared.FetchSubs(m.profiles, m.subs),
			)
		}
		if msg.Op == "sub-remove" {
			delete(m.errs, msg.Label)
			return m, tea.Batch(
				m.showToast("Удалена: "+msg.Label, true),
				shared.FetchSubs(m.profiles, m.subs),
			)
		}
		if msg.Op == "sub-edit" {
			delete(m.errs, msg.Label)
			return m, tea.Batch(
				m.showToast("Изменена: "+msg.Label, true),
				shared.FetchSubs(m.profiles, m.subs),
			)
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
	case "a":
		if m.busy {
			return m, nil
		}
		m.inputStep = 1
		m.tmpName = ""
		m.input.Open("Имя подписки:", "")
		return m, nil
	case "e":
		if m.busy || !m.loaded || len(m.list) == 0 {
			return m, nil
		}
		sub := m.list[m.cursor]
		m.pending = sub
		m.editStep = 1
		m.input.Open("Имя подписки:", sub.Name)
		return m, nil
	case "x":
		if m.busy || !m.loaded || len(m.list) == 0 {
			return m, nil
		}
		sub := m.list[m.cursor]
		m.pending = sub
		m.confirm.Ask(fmt.Sprintf("Удалить %s?", sub.Name), "delete")
		return m, nil
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

// updateInput drives the two-step add form: name, then URL.
func (m *Model) updateInput(msg tea.Msg) (shared.Screen, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch kp.String() {
	case "enter":
		if m.inputStep == 1 {
			m.tmpName = m.input.Value()
			if m.tmpName == "" {
				return m, nil
			}
			m.inputStep = 2
			m.input.Open("URL подписки:", "")
			return m, nil
		}
		url := m.input.Value()
		m.inputStep = 0
		m.input.Close()
		if url == "" {
			return m, nil
		}
		m.busy = true
		return m, shared.DoSubAdd(m.subs, m.tmpName, url)
	case "esc":
		m.inputStep = 0
		m.input.Close()
		return m, nil
	default:
		m.input.Key(kp.String())
		return m, nil
	}
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
		_, ok := m.confirm.Resolve()
		if !ok {
			return m, nil
		}
		m.busy = true
		return m, shared.DoSubRemove(m.subs, m.pending.ID, m.pending.Name)
	case "esc", "q":
		m.confirm.Resolve()
		return m, nil
	}
	return m, nil
}

// updateEdit drives the two-step edit form: name, then URL.
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
			m.input.Open("URL подписки:", m.pending.URL)
			return m, nil
		}
		url := m.input.Value()
		m.editStep = 0
		m.input.Close()
		if url == "" {
			url = m.pending.URL
		}
		m.busy = true
		return m, shared.DoSubEdit(m.subs, m.pending.ID, m.editName, url, m.pending.Name)
	case "esc":
		m.editStep = 0
		m.input.Close()
		return m, nil
	default:
		m.input.Key(kp.String())
		return m, nil
	}
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
		b.WriteString(m.styles.FieldLabel.Render(fmt.Sprintf("%-22s %10s  %s", "NAME", "PROFILES", "UPDATED")) + "\n")
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
		row := fmt.Sprintf("%s%s %3d profiles  %s",
			cursor, m.styles.FieldLabel.Render(fmt.Sprintf("%-20s", name)),
			m.counts[domain.SubscriptionSource(s.ID)], updated)
		if i == m.cursor {
			row = m.styles.SelectedRow.Render(row)
		}
		b.WriteString(row + "\n")
		if emsg, ok := m.errs[s.ID]; ok {
			b.WriteString("  " + m.styles.Err.Render("! "+emsg) + "\n")
		}
	}
	if m.errText != "" {
		b.WriteString(m.styles.Err.Render(m.errText) + "\n")
	}
	if t := m.toast.View(); t != "" {
		b.WriteString(t + "\n")
	}
	if m.inputStep != 0 {
		b.WriteString("\n" + m.input.View())
	}
	out := shared.IndentLines(b.String(), " ")
	// the confirm box hugs the left edge, outside the page indent
	if m.confirm.Showing() {
		out += "\n" + m.confirm.View()
	}
	return out
}
