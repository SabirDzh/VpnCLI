package shared

import (
	"context"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Screen is implemented by every TUI tab. View renders into the given
// size only; the root model assembles the final tea.View.
type Screen interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Screen, tea.Cmd)
	View(width, height int) string
	Title() string
	Keys() []key.Binding
}

const opTimeout = 30 * time.Second

// withTimeout runs fn with a bounded context, keeping Update non-blocking.
func withTimeout(parent context.Context, fn func(ctx context.Context) tea.Msg) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(parent, opTimeout)
		defer cancel()
		return fn(ctx)
	}
}

// FetchStatus polls the connection status once.
func FetchStatus(ctx context.Context, api ConnectionAPI) tea.Cmd {
	return withTimeout(ctx, func(ctx context.Context) tea.Msg {
		st, err := api.Status(ctx)
		return StatusMsg{St: st, Err: err}
	})
}

// ScheduleTick plans the next poll tick.
func ScheduleTick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return TickMsg{} })
}

// Back returns to the main menu.
func Back() tea.Cmd {
	return func() tea.Msg { return BackMsg{} }
}

// IndentLines prefixes every line of s (ANSI-safe: prefix added raw).
func IndentLines(s, prefix string) string {
	if s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = prefix + l
		}
	}
	return strings.Join(lines, "\n")
}

// Truncate shortens s to at most n runes, adding … on cut.
func Truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

// DoUp connects the given profile.
func DoUp(ctx context.Context, api ConnectionAPI, ref, label string) tea.Cmd {
	return withTimeout(ctx, func(ctx context.Context) tea.Msg {
		return OpDoneMsg{Op: "up", Label: label, Err: api.Up(ctx, ref)}
	})
}

// DoDown disconnects the VPN.
func DoDown(ctx context.Context, api ConnectionAPI) tea.Cmd {
	return withTimeout(ctx, func(ctx context.Context) tea.Msg {
		return OpDoneMsg{Op: "down", Err: api.Down(ctx)}
	})
}

// FetchProfiles loads profiles with the active id.
func FetchProfiles(profiles ProfileAPI) tea.Cmd {
	return func() tea.Msg {
		list, err := profiles.List()
		if err != nil {
			return ProfilesMsg{Err: err}
		}
		activeID := ""
		if a, err := profiles.Active(); err == nil {
			activeID = a.ID
		}
		return ProfilesMsg{List: list, ActiveID: activeID}
	}
}

// DoUse selects the active profile.
func DoUse(profiles ProfileAPI, idOrName, label string) tea.Cmd {
	return func() tea.Msg {
		_, err := profiles.Use(idOrName)
		return OpDoneMsg{Op: "use", Label: label, Err: err}
	}
}

// DoAdd imports a profile from a URI string.
func DoAdd(profiles ProfileAPI, uri string) tea.Cmd {
	return func() tea.Msg {
		p, err := profiles.Add(uri)
		if err != nil {
			return OpDoneMsg{Op: "add", Err: err}
		}
		return OpDoneMsg{Op: "add", Label: p.Name}
	}
}

// DoRemove deletes a profile.
func DoRemove(profiles ProfileAPI, id, label string) tea.Cmd {
	return func() tea.Msg {
		return OpDoneMsg{Op: "remove", Label: label, Err: profiles.Remove(id)}
	}
}

// FetchSubs loads subscriptions with profile counts.
func FetchSubs(profiles ProfileAPI, subs SubscriptionAPI) tea.Cmd {
	return func() tea.Msg {
		list, err := subs.List()
		if err != nil {
			return SubsMsg{Err: err}
		}
		counts := map[string]int{}
		if pl, err := profiles.List(); err == nil {
			for _, p := range pl {
				counts[p.Source]++
			}
		}
		return SubsMsg{Subs: list, Counts: counts}
	}
}

// DoSubAdd registers a subscription source.
func DoSubAdd(subs SubscriptionAPI, name, url string) tea.Cmd {
	return func() tea.Msg {
		s, err := subs.Add(name, url)
		if err != nil {
			return OpDoneMsg{Op: "sub-add", Err: err}
		}
		return OpDoneMsg{Op: "sub-add", Label: s.Name}
	}
}

// DoSubRemove deletes a subscription source.
func DoSubRemove(subs SubscriptionAPI, id, label string) tea.Cmd {
	return func() tea.Msg {
		return OpDoneMsg{Op: "sub-remove", Label: label, Err: subs.Remove(id)}
	}
}

// DoSubUpdate refreshes one (or all) subscriptions.
func DoSubUpdate(subs SubscriptionAPI, idOrName string, all bool) tea.Cmd {
	op := "update"
	if all {
		op = "update-all"
	}
	return withTimeout(context.Background(), func(ctx context.Context) tea.Msg {
		if all {
			list, err := subs.List()
			if err != nil {
				return OpDoneMsg{Op: op, Err: err}
			}
			total := 0
			for _, s := range list {
				n, err := subs.Update(s.ID)
				if err != nil {
					return OpDoneMsg{Op: op, Label: s.Name, Err: err}
				}
				total += n
			}
			return OpDoneMsg{Op: op, N: total}
		}
		n, err := subs.Update(idOrName)
		return OpDoneMsg{Op: op, Label: idOrName, N: n, Err: err}
	})
}
