// Package status implements the Status tab: connection state, active
// profile, core, uptime and connect/disconnect actions.
package status

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/tui/component"
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

// Model is the Status tab state.
type Model struct {
	api      shared.ConnectionAPI
	styles   theme.Styles
	readOnly bool
	ctx      context.Context

	st      connState
	errText string
	busy    bool
	busyOp  string
	spin    spinner.Model
	toast   component.Toast
	width   int
}

// connState is the rendered subset of app.StatusView.
type connState struct {
	running  bool
	name     string
	core     string
	pid      int
	since    time.Time
	endpoint string
}

// New creates the Status tab.
func New(ctx context.Context, api shared.ConnectionAPI, st theme.Styles, readOnly bool) *Model {
	return &Model{
		api:      api,
		styles:   st,
		readOnly: readOnly,
		ctx:      ctx,
		spin:     spinner.New(spinner.WithSpinner(spinner.Dot)),
		toast:    component.NewToast(st),
	}
}

// Title implements shared.Screen.
func (m *Model) Title() string { return "Status" }

// Keys implements shared.Screen.
func (m *Model) Keys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter", "c"), key.WithHelp("enter/c", "connect/disconnect")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

// Init implements shared.Screen.
func (m *Model) Init() tea.Cmd {
	return shared.FetchStatus(m.ctx, m.api)
}

// Update implements shared.Screen.
func (m *Model) Update(msg tea.Msg) (shared.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case shared.StatusMsg:
		if msg.Err != nil {
			m.errText = shared.DescribeError(msg.Err)
			m.st = connState{}
		} else {
			m.errText = ""
			m.st = connState{
				running:  msg.St.Running,
				name:     msg.St.ProfileName,
				core:     msg.St.Core,
				pid:      msg.St.PID,
				since:    msg.St.Since,
				endpoint: msg.St.Endpoint,
			}
		}
		return m, shared.ScheduleTick(2 * time.Second)
	case shared.TickMsg:
		return m, shared.FetchStatus(m.ctx, m.api)
	case shared.OpDoneMsg:
		m.busy = false
		m.busyOp = ""
		if msg.Err != nil {
			return m, m.showToast(shared.DescribeError(msg.Err), false)
		}
		verb := map[string]string{"up": "Подключено", "down": "Отключено"}[msg.Op]
		if verb == "" {
			verb = "Готово"
		}
		if msg.Label != "" {
			verb += ": " + msg.Label
		}
		return m, tea.Batch(
			m.showToast(verb, true),
			shared.FetchStatus(m.ctx, m.api),
		)
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
	}
	return m, nil
}

func (m *Model) onKey(k string) (shared.Screen, tea.Cmd) {
	switch k {
	case "r":
		if m.busy {
			return m, nil
		}
		return m, shared.FetchStatus(m.ctx, m.api)
	case "enter", "c":
		return m.toggle()
	case "esc":
		return m, shared.Back()
	}
	return m, nil
}

// toggle connects or disconnects. Double presses in busy are ignored.
func (m *Model) toggle() (shared.Screen, tea.Cmd) {
	if m.busy {
		return m, nil
	}
	if m.readOnly {
		return m, m.showToast(shared.DescribeError(domain.ErrNotPrivileged), false)
	}
	m.busy = true
	if m.st.running {
		m.busyOp = "down"
		return m, tea.Batch(m.spin.Tick, shared.DoDown(m.ctx, m.api))
	}
	if m.st.name == "" {
		m.busy = false
		return m, m.showToast("Профиль не выбран. Вкладка Profiles + enter", false)
	}
	m.busyOp = "up"
	label := m.st.name
	return m, tea.Batch(m.spin.Tick, shared.DoUp(m.ctx, m.api, "", label))
}

func (m *Model) showToast(text string, ok bool) tea.Cmd {
	return m.toast.Show(text, ok, func(id int) tea.Msg {
		return shared.ToastExpiredMsg{ID: id}
	})
}

// View implements shared.Screen.
func (m *Model) View(width, height int) string {
	m.width = width
	var b strings.Builder
	b.WriteString(m.styles.Title.Render("Status") + "\n\n")
	switch {
	case m.busy:
		fmt.Fprintf(&b, "%s %s…\n", m.spin.View(), m.busyOp)
		b.WriteString(m.styles.Dim.Render("операция выполняется, подожди") + "\n")
	case m.st.running:
		b.WriteString(m.styles.OK.Bold(true).Render("● connected") + "\n\n")
		b.WriteString(m.infoPanel() + "\n")
	default:
		b.WriteString(m.styles.Dim.Render("○ disconnected") + "\n")
		// an active profile exists: show what enter would dial
		if m.st.name != "" {
			b.WriteString("\n" + m.infoPanel() + "\n")
		}
	}
	b.WriteString("\n")
	if m.errText != "" {
		b.WriteString(m.styles.Err.Render("! "+m.errText) + "\n\n")
	}
	if t := m.toast.View(); t != "" {
		b.WriteString(t + "\n")
	}
	return shared.IndentLines(b.String(), " ")
}

// infoPanel renders the bordered connection summary rows.
func (m *Model) infoPanel() string {
	row := func(label, value string) string {
		return m.styles.Label.Render(label) + m.styles.Value.Render(value)
	}
	var rows []string
	if m.st.running {
		rows = append(rows,
			row("profile", m.st.name),
			row("core", fmt.Sprintf("%s · pid %d", m.st.core, m.st.pid)),
		)
		if !m.st.since.IsZero() {
			rows = append(rows, row("uptime", time.Since(m.st.since).Round(time.Second).String()))
		}
		if m.st.endpoint != "" {
			rows = append(rows, row("endpoint", m.st.endpoint))
		}
	} else {
		rows = append(rows, row("profile", m.st.name))
		if m.st.endpoint != "" {
			rows = append(rows, row("endpoint", m.st.endpoint))
		}
	}
	return m.styles.Box.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
