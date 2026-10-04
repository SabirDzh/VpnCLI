// Package other implements the "Other" tab: about info, update checks
// and installs, auto-update toggle and the repository link.
package other

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/app"
	"github.com/SabirDzh/VpnCLI/internal/platform"
	"github.com/SabirDzh/VpnCLI/internal/tui/component"
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

// checkDoneMsg resolves an update check.
type checkDoneMsg struct {
	res app.CheckResult
	err error
}

// updateDoneMsg resolves an update install.
type updateDoneMsg struct {
	tag string
	err error
}

// Model is the Other tab state.
type Model struct {
	deps   shared.Deps
	styles theme.Styles
	toast  component.Toast
	// openURL is overridable for tests.
	openURL func(string) error
	latest  string
	auto    bool
	width   int
}

// New creates the Other tab.
func New(deps shared.Deps, st theme.Styles) *Model {
	auto := deps.Settings.AutoUpdate
	if deps.SettingsAPI != nil {
		auto = deps.SettingsAPI.Snapshot().AutoUpdate
	}
	return &Model{
		deps:    deps,
		styles:  st,
		toast:   component.NewToast(st),
		openURL: platform.OpenBrowser,
		auto:    auto,
	}
}

// Title implements shared.Screen.
func (m *Model) Title() string { return "Other" }

// Keys implements shared.Screen.
func (m *Model) Keys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "check updates")),
		key.NewBinding(key.WithKeys("U"), key.WithHelp("U", "install update")),
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "auto-updates on/off")),
		key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open repository")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

// Init implements shared.Screen.
func (m *Model) Init() tea.Cmd { return nil }

// Update implements shared.Screen.
func (m *Model) Update(msg tea.Msg) (shared.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case checkDoneMsg:
		if msg.err != nil {
			return m, m.showToast("Ошибка: "+msg.err.Error(), false)
		}
		m.latest = msg.res.Latest
		if msg.res.Available {
			return m, m.showToast("Доступно: "+msg.res.Latest+" — U чтобы обновить", true)
		}
		return m, m.showToast("Актуально: "+msg.res.Latest, true)
	case updateDoneMsg:
		if msg.err != nil {
			return m, m.showToast("Ошибка: "+msg.err.Error(), false)
		}
		return m, m.showToast("Обновлено до "+msg.tag+" — перезапусти vpn", true)
	case shared.ToastExpiredMsg:
		m.toast.Expire(msg.ID)
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyPressMsg:
		return m.onKey(msg.String())
	}
	return m, nil
}

func (m *Model) onKey(k string) (shared.Screen, tea.Cmd) {
	switch k {
	case "c":
		if m.deps.Update == nil {
			return m, nil
		}
		api := m.deps.Update
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			res, err := api.Check(ctx)
			return checkDoneMsg{res: res, err: err}
		}
	case "U":
		if m.deps.Update == nil || m.latest == "" {
			return m, nil
		}
		api := m.deps.Update
		tag := m.latest
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			return updateDoneMsg{tag: tag, err: api.Update(ctx, tag)}
		}
	case "a":
		if m.deps.SettingsAPI == nil {
			return m, nil
		}
		if err := m.deps.SettingsAPI.SetAutoUpdate(!m.auto); err != nil {
			return m, m.showToast("Ошибка: "+err.Error(), false)
		}
		m.auto = !m.auto
		return m, m.showToast("Сохранено", true)
	case "o":
		url := repoURL(m.deps.Repo)
		if url == "" {
			return m, nil
		}
		if err := m.openURL(url); err != nil {
			return m, m.showToast("Ошибка: "+err.Error(), false)
		}
		return m, m.showToast("Открываю "+url, true)
	case "esc":
		return m, shared.Back()
	}
	return m, nil
}

func repoURL(repo string) string {
	if repo == "" {
		return ""
	}
	return "https://github.com/" + repo
}

func (m *Model) showToast(text string, ok bool) tea.Cmd {
	return m.toast.Show(text, ok, func(id int) tea.Msg {
		return shared.ToastExpiredMsg{ID: id}
	})
}

// View implements shared.Screen.
func (m *Model) View(width, height int) string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render("Other") + "\n\n")
	b.WriteString(m.styles.Dim.Render("── About ──") + "\n")
	row := func(label, value string) {
		b.WriteString("  " + fmt.Sprintf("%-14s %s", label, m.styles.Value.Render(value)) + "\n")
	}
	ver := m.deps.Version
	if ver == "" {
		ver = "dev"
	}
	row("version", ver)
	row("repository", repoURL(m.deps.Repo))
	row("developer", "Sabir Dzhabrailov")
	b.WriteString("\n" + m.styles.Dim.Render("── Updates ──") + "\n")
	latest := m.latest
	if latest == "" {
		latest = "press c to check"
	}
	row("latest", latest)
	row("last check", dateOrNever(m.deps.Update))
	row("last update", dateOrNever2(m.deps.Update))
	row("auto-updates", onOff(m.auto))
	b.WriteString("\n" + m.styles.Dim.Render("автообновление проверяет релизы при каждом запуске vpn (cli и tui)") + "\n")
	if t := m.toast.View(); t != "" {
		b.WriteString("\n" + t + "\n")
	}
	_ = width
	_ = height
	return shared.IndentLines(b.String(), " ")
}

func dateOrNever(u shared.UpdateAPI) string {
	if u == nil || u.LastCheck().IsZero() {
		return "never"
	}
	return u.LastCheck().Format("02.01 15:04")
}

func dateOrNever2(u shared.UpdateAPI) string {
	if u == nil || u.LastUpdated().IsZero() {
		return "never"
	}
	return u.LastUpdated().Format("02.01 15:04")
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
