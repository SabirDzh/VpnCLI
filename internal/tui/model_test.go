package tui

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/app"
	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/tui/screen/profiles"
	"github.com/SabirDzh/VpnCLI/internal/tui/screen/status"
	"github.com/SabirDzh/VpnCLI/internal/tui/screen/subscriptions"
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
	"github.com/SabirDzh/VpnCLI/internal/tui/testutil"
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

func strip(s string) string { return ansiRe.ReplaceAllString(s, "") }

func testDeps() (shared.Deps, *testutil.FakeConn, *testutil.FakeProfiles, *testutil.FakeSubs) {
	conn := &testutil.FakeConn{St: app.StatusView{Running: true, Core: "sing-box", ProfileName: "home", PID: 123, Since: time.Now().Add(-time.Hour)}}
	prof := &testutil.FakeProfiles{
		Items: []domain.Profile{
			{ID: "a1", Name: "home", Protocol: domain.ProtocolVLESS, Source: domain.ManualSource},
			{ID: "b2", Name: "work", Protocol: domain.ProtocolTrojan, Source: "subscription:s1"},
		},
		ActiveID: "a1",
	}
	subs := &testutil.FakeSubs{Items: []domain.Subscription{{ID: "s1", Name: "sub"}}}
	return shared.Deps{Connection: conn, Profiles: prof, Subs: subs}, conn, prof, subs
}

func sized(m *Model, w, h int) *Model {
	nm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return nm.(*Model)
}

func TestMenuNavigation(t *testing.T) {
	deps, _, _, _ := testDeps()
	m := sized(NewModel(context.Background(), deps), 80, 24)
	if m.page != nil || m.cursor != 0 {
		t.Fatal("must start on the main menu")
	}
	nm, _ := m.Update(keyPress("j"))
	m = nm.(*Model)
	if m.cursor != 1 {
		t.Fatal("j must move down")
	}
	nm, _ = m.Update(keyPress("k"))
	m = nm.(*Model)
	if m.cursor != 0 {
		t.Fatal("k must move up")
	}
	// number keys open pages directly
	nm, _ = m.Update(keyPress("2"))
	m = nm.(*Model)
	if m.page == nil || m.page.Title() != "Profiles" {
		t.Fatal("2 must open Profiles")
	}
	nm, _ = m.Update(shared.BackMsg{})
	m = nm.(*Model)
	// enter opens the page, esc returns (cursor still on Profiles)
	nm, cmd := m.Update(keyPress("enter"))
	m = nm.(*Model)
	if m.page == nil {
		t.Fatal("enter must open a page")
	}
	if cmd == nil {
		t.Fatal("opening a page must fetch")
	}
	if m.page.Title() != "Profiles" {
		t.Fatalf("want Profiles, got %s", m.page.Title())
	}
	nm, _ = m.Update(shared.BackMsg{})
	m = nm.(*Model)
	if m.page != nil {
		t.Fatal("BackMsg must return to menu")
	}
}

func TestPageEscGoesBack(t *testing.T) {
	deps, _, _, _ := testDeps()
	m := sized(NewModel(context.Background(), deps), 80, 24)
	nm, _ := m.Update(keyPress("enter")) // open Status
	m = nm.(*Model)
	if m.page == nil {
		t.Fatal("must open status page")
	}
	nm, cmd := m.Update(keyPress("esc")) // status emits Back
	m = nm.(*Model)
	_ = cmd
	if cmd == nil {
		t.Fatal("esc must produce a command")
	}
	msg := cmd()
	if _, ok := msg.(shared.BackMsg); !ok {
		t.Fatalf("esc must request back, got %T", msg)
	}
	nm, _ = m.Update(msg)
	if nm.(*Model).page != nil {
		t.Fatal("must be back on menu")
	}
}

func TestQuit(t *testing.T) {
	deps, _, _, _ := testDeps()
	m := sized(NewModel(context.Background(), deps), 80, 24)
	_, cmd := m.Update(keyPress("ctrl+c"))
	if cmd == nil {
		t.Fatal("q must return a command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("q must quit, got %T", cmd())
	}
}

func TestMinSize(t *testing.T) {
	deps, _, _, _ := testDeps()
	m := sized(NewModel(context.Background(), deps), 40, 10)
	if !strings.Contains(strip(m.render()), "маленькое") {
		t.Fatal("must warn on small window")
	}
	m2 := sized(NewModel(context.Background(), deps), 80, 24)
	if !strings.Contains(strip(m2.render()), "Status") {
		t.Fatal("must render tabs on normal window")
	}
}

func TestStatusError(t *testing.T) {
	deps, conn, _, _ := testDeps()
	conn.StatusErr = domain.ErrNotPrivileged
	st := status.New(context.Background(), deps.Connection, theme.Default(), true)
	ns, _ := st.Update(shared.StatusMsg{Err: domain.ErrNotPrivileged})
	s := ns.(*status.Model)
	if out := strip(s.View(80, 24)); !strings.Contains(out, "sudo vpn tui") {
		t.Fatalf("must explain privileges, got:\n%s", out)
	}
}

func TestStatusBusyIgnoresDoubleToggle(t *testing.T) {
	deps, conn, _, _ := testDeps()
	st := status.New(context.Background(), deps.Connection, theme.Default(), false)
	ns, _ := st.Update(shared.StatusMsg{St: app.StatusView{Running: false, ProfileName: "home"}})
	ns2, cmd2 := ns.Update(keyPress("enter")) // toggles up, busy now
	if cmd2 == nil {
		t.Fatal("first toggle must start an operation")
	}
	// run the batched cmds: spinner tick is a no-op here, DoUp records the call
	if batch, ok := cmd2().(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				_ = c()
			}
		}
	} else {
		_ = cmd2()
	}
	ns3, cmd3 := ns2.Update(keyPress("enter")) // ignored while busy
	_ = ns3
	if cmd3 != nil {
		t.Fatal("second toggle while busy must be ignored")
	}
	if len(conn.UpCalls) != 1 {
		t.Fatalf("Up called %d times", len(conn.UpCalls))
	}
}

func TestDescribeError(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{domain.ErrNotPrivileged, "sudo vpn tui"},
		{domain.ErrNoActiveProfile, "Profiles"},
		{domain.ErrCoreNotFound, "sing-box"},
		{domain.ErrAlreadyRunning, "уже подключён"},
		{context.DeadlineExceeded, "время ожидания"},
	}
	for _, c := range cases {
		if got := shared.DescribeError(c.err); !strings.Contains(got, c.want) {
			t.Errorf("DescribeError(%v) = %q, want %q", c.err, got, c.want)
		}
	}
}

func TestProfilesItemsAndActive(t *testing.T) {
	deps, _, _, _ := testDeps()
	p := profiles.New(context.Background(), deps.Connection, deps.Profiles, theme.Default(), false)
	ns, _ := p.Update(shared.ProfilesMsg{
		List: []domain.Profile{
			{ID: "a1", Name: "home", Protocol: "vless", Source: "manual"},
		},
		ActiveID: "a1",
	})
	out := strip(ns.(*profiles.Model).View(80, 20))
	if !strings.Contains(out, "home") || !strings.Contains(out, "*") {
		t.Fatalf("must list profile with active marker, got:\n%s", out)
	}
	if strings.Contains(out, "203.0.113") {
		t.Fatal("hosts must not leak into the list")
	}
}

func TestSubsCounts(t *testing.T) {
	deps, _, _, _ := testDeps()
	s := subscriptions.New(deps, theme.Default())
	ns, _ := s.Update(shared.SubsMsg{
		Subs:   []domain.Subscription{{ID: "s1", Name: "sub"}},
		Counts: map[string]int{"subscription:s1": 3},
	})
	out := strip(ns.(*subscriptions.Model).View(80, 20))
	if !strings.Contains(out, "3 profiles") {
		t.Fatalf("must show counts, got:\n%s", out)
	}
}

func keyPress(s string) tea.Msg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	default:
		return tea.KeyPressMsg{Code: rune(s[0])}
	}
}

func TestHelpOverlay(t *testing.T) {
	deps, _, _, _ := testDeps()
	m := sized(NewModel(context.Background(), deps), 80, 24)
	nm, _ := m.Update(keyPress("?"))
	m = nm.(*Model)
	if !strings.Contains(strip(m.render()), "Клавиши") {
		t.Fatal("must show full help")
	}
	nm, _ = m.Update(keyPress("?"))
	m = nm.(*Model)
	if strings.Contains(strip(m.render()), "Клавиши") {
		t.Fatal("must hide full help")
	}
	// number key opens the page with a fetch command
	nm, cmd := m.Update(keyPress("1"))
	m = nm.(*Model)
	if m.page == nil || m.page.Title() != "Status" {
		t.Fatal("1 must open Status")
	}
	if cmd == nil {
		t.Fatal("opening must fetch")
	}
}

func TestNotReadyRendersLoading(t *testing.T) {
	deps, _, _, _ := testDeps()
	m := NewModel(context.Background(), deps) // no size yet
	if strip(m.render()) != "loading…" {
		t.Fatalf("got %q", m.render())
	}
}

func TestHeaderFooter(t *testing.T) {
	deps, _, _, _ := testDeps()
	m := sized(NewModel(context.Background(), deps), 80, 24)
	out := strip(m.render())
	for _, want := range []string{"╱", "Profiles", "quit", "navigate"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestMenuSummaries(t *testing.T) {
	deps, _, _, _ := testDeps()
	m := sized(NewModel(context.Background(), deps), 80, 24)
	// feed summaries
	nm, _ := m.Update(shared.StatusMsg{St: app.StatusView{Running: true, ProfileName: "home"}})
	m = nm.(*Model)
	nm, _ = m.Update(shared.ProfilesMsg{
		List:     []domain.Profile{{ID: "a1", Name: "home"}},
		ActiveID: "a1",
	})
	m = nm.(*Model)
	nm, _ = m.Update(shared.SubsMsg{Subs: []domain.Subscription{{ID: "s1"}}})
	m = nm.(*Model)
	out := strip(m.render())
	for _, want := range []string{"home", "1 profiles", "1 subs"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	// space opens too
	nm, _ = m.Update(keyPress("space"))
	if nm.(*Model).page == nil {
		t.Fatal("space must open a page")
	}
}

func TestMenuErrorSummary(t *testing.T) {
	deps, _, _, _ := testDeps()
	m := sized(NewModel(context.Background(), deps), 80, 24)
	nm, _ := m.Update(shared.StatusMsg{Err: domain.ErrNotRunning})
	out := strip(nm.(*Model).render())
	if !strings.Contains(out, "error") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestNumberKeysOpenPages(t *testing.T) {
	deps, _, _, _ := testDeps()
	m := sized(NewModel(context.Background(), deps), 80, 24)
	titles := []string{"Status", "Profiles", "Subscriptions"}
	for i, key := range []string{"1", "2", "3"} {
		nm, cmd := m.Update(keyPress(key))
		m = nm.(*Model)
		if m.page == nil || m.page.Title() != titles[i] {
			t.Fatalf("key %s must open %s", key, titles[i])
		}
		if cmd == nil {
			t.Fatal("opening must fetch")
		}
		nm, _ = m.Update(shared.BackMsg{})
		m = nm.(*Model)
	}
}
