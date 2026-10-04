package subscriptions

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
	"github.com/SabirDzh/VpnCLI/internal/tui/testutil"
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

func testModel() (*Model, *testutil.FakeSubs) {
	subs := &testutil.FakeSubs{
		Items: []domain.Subscription{
			{ID: "s1", Name: "one"},
			{ID: "s2", Name: "two"},
		},
		UpdateN: 5,
	}
	prof := &testutil.FakeProfiles{Items: []domain.Profile{
		{ID: "p1", Source: "subscription:s1"},
	}}
	m := New(shared.Deps{Profiles: prof, Subs: subs}, theme.Default())
	ns, _ := m.Update(shared.SubsMsg{Subs: subs.Items, Counts: map[string]int{"subscription:s1": 1}})
	return ns.(*Model), subs
}

func kf(c byte) tea.Msg { return tea.KeyPressMsg{Code: rune(c)} }

// runCmd executes a command and, for batches, every nested command.
func runCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				_ = c()
			}
		}
	}
}

func TestCursorAndUpdate(t *testing.T) {
	m, subs := testModel()
	if m.cursor != 0 {
		t.Fatal("cursor starts at 0")
	}
	ns, _ := m.Update(kf('j'))
	if ns.(*Model).cursor != 1 {
		t.Fatal("j must move down")
	}
	_, cmd := ns.Update(kf('u'))
	if cmd == nil {
		t.Fatal("u must start update")
	}
	runCmd(cmd)
	if len(subs.Updated) != 1 || subs.Updated[0] != "s2" {
		t.Fatalf("Updated = %v", subs.Updated)
	}
}

func TestUpdateAllAndDone(t *testing.T) {
	m, subs := testModel()
	ns, cmd := m.Update(kf('U'))
	if cmd == nil {
		t.Fatal("U must start update-all")
	}
	runCmd(cmd)
	if len(subs.Updated) != 2 {
		t.Fatalf("must update both, got %v", subs.Updated)
	}
	ns2, _ := ns.Update(shared.OpDoneMsg{Op: "update-all", N: 10})
	out := ns2.(*Model).toast.View()
	if !strings.Contains(out, "10") {
		t.Fatalf("must toast total, got %q", out)
	}
}

func TestUpdateErrorToast(t *testing.T) {
	m, subs := testModel()
	subs.UpdateErr = errBoom{}
	ns, cmd := m.Update(kf('u'))
	runCmd(cmd)
	ns2, _ := ns.Update(shared.OpDoneMsg{Op: "update", Label: "one", Err: errBoom{}})
	out := ns2.(*Model).toast.View()
	if !strings.Contains(out, "boom") {
		t.Fatalf("must toast error, got %q", out)
	}
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }

func TestRefreshEmptyAndError(t *testing.T) {
	subs := &testutil.FakeSubs{}
	prof := &testutil.FakeProfiles{}
	m := New(shared.Deps{Profiles: prof, Subs: subs}, theme.Default())
	// loading state before first message
	if out := m.View(80, 20); !strings.Contains(out, "loading") {
		t.Fatalf("got:\n%s", out)
	}
	ns, _ := m.Update(shared.SubsMsg{})
	out := ns.(*Model).View(80, 20)
	if !strings.Contains(out, "vpn sub add") {
		t.Fatalf("must hint add command, got:\n%s", out)
	}
	// error state
	ns2, _ := ns.Update(shared.SubsMsg{Err: errBoom{}})
	if out := ns2.(*Model).View(80, 20); !strings.Contains(out, "boom") {
		t.Fatalf("got:\n%s", out)
	}
	// busy ignores update keys
	busy := ns2.(*Model)
	busy.busy = true
	if _, cmd := busy.Update(kf('u')); cmd != nil {
		t.Fatal("must ignore u while busy")
	}
	if _, cmd := busy.Update(kf('r')); cmd != nil {
		t.Fatal("must ignore r while busy")
	}
	// cursor clamps after shrink
	ns3, _ := busy.Update(shared.SubsMsg{Subs: []domain.Subscription{{ID: "s1"}}})
	if ns3.(*Model).cursor != 0 {
		t.Fatal("cursor must clamp")
	}
}

func TestKeysAndSpinner(t *testing.T) {
	m, _ := testModel()
	// k moves up
	ns, _ := m.Update(kf('j'))
	ns, _ = ns.Update(kf('k'))
	if ns.(*Model).cursor != 0 {
		t.Fatal("k must move up")
	}
	// down at bottom stays
	ns, _ = ns.Update(kf('j'))
	ns, _ = ns.Update(kf('j'))
	if ns.(*Model).cursor != 1 {
		t.Fatal("cursor must clamp at bottom")
	}
	// window size stored
	ns, _ = ns.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if ns.(*Model).width != 100 {
		t.Fatal("must store width")
	}
	// spinner ticks while busy
	busy := ns.(*Model)
	busy.busy = true
	if _, cmd := busy.Update(spinnerTickMsg()); cmd == nil {
		t.Fatal("spinner must tick while busy")
	}
}

func spinnerTickMsg() tea.Msg { return spinner.TickMsg{} }

func TestEmptyListKeysIgnored(t *testing.T) {
	subs := &testutil.FakeSubs{}
	prof := &testutil.FakeProfiles{}
	m := New(shared.Deps{Profiles: prof, Subs: subs}, theme.Default())
	ns, _ := m.Update(shared.SubsMsg{})
	for _, k := range []byte{'u', 'U', 'j', 'k'} {
		if _, cmd := ns.Update(kf(k)); cmd != nil {
			t.Fatalf("key %c on empty list must be silent", k)
		}
	}
}
