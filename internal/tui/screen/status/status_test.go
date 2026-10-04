package status

import (
	"context"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"

	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/app"
	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
	"github.com/SabirDzh/VpnCLI/internal/tui/testutil"
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

func kf(s string) tea.Msg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "r":
		return tea.KeyPressMsg{Code: 'r'}
	case "c":
		return tea.KeyPressMsg{Code: 'c'}
	default:
		return tea.KeyPressMsg{Code: rune(s[0])}
	}
}

func TestToggleUpFlow(t *testing.T) {
	conn := &testutil.FakeConn{}
	st := New(context.Background(), conn, theme.Default(), false)
	ns, _ := st.Update(shared.StatusMsg{St: app.StatusView{Running: false, ProfileName: "home"}})
	ns2, cmd := ns.Update(kf("enter"))
	if cmd == nil {
		t.Fatal("must start up operation")
	}
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				_ = c()
			}
		}
	}
	if len(conn.UpCalls) != 1 {
		t.Fatalf("Up calls = %d", len(conn.UpCalls))
	}
	// busy ignores further toggles
	if _, cmd := ns2.Update(kf("c")); cmd != nil {
		t.Fatal("must ignore toggle while busy")
	}
}

func TestToggleDownFlow(t *testing.T) {
	conn := &testutil.FakeConn{}
	st := New(context.Background(), conn, theme.Default(), false)
	ns, _ := st.Update(shared.StatusMsg{St: app.StatusView{Running: true, ProfileName: "home", Core: "sing-box"}})
	ns2, cmd := ns.Update(kf("c"))
	if cmd == nil {
		t.Fatal("must start down operation")
	}
	_ = ns2
	// complete with error -> toast, not busy
	ns3, _ := ns2.Update(shared.OpDoneMsg{Op: "down", Err: domain.ErrNotRunning})
	out := stripAnsi(ns3.(*Model).View(80, 24))
	if !strings.Contains(out, "уже отключён") {
		t.Fatalf("must show error toast, got:\n%s", out)
	}
}

func TestReadOnlyToggle(t *testing.T) {
	conn := &testutil.FakeConn{}
	st := New(context.Background(), conn, theme.Default(), true)
	ns, _ := st.Update(shared.StatusMsg{St: app.StatusView{}})
	ns2, _ := ns.Update(kf("enter"))
	out := stripAnsi(ns2.(*Model).View(80, 24))
	if !strings.Contains(out, "sudo vpn tui") {
		t.Fatalf("must refuse in readonly, got:\n%s", out)
	}
	if len(conn.UpCalls) != 0 {
		t.Fatal("must not call Up in readonly")
	}
}

func stripAnsi(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc && (r == 'm' || ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z')):
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestViewStates(t *testing.T) {
	conn := &testutil.FakeConn{}
	st := New(context.Background(), conn, theme.Default(), false)
	// disconnected, no data
	if out := stripAnsi(st.View(80, 24)); !strings.Contains(out, "disconnected") {
		t.Fatalf("got:\n%s", out)
	}
	// connected full
	ns, _ := st.Update(shared.StatusMsg{St: app.StatusView{
		Running: true, ProfileName: "home", Core: "sing-box",
		PID: 1, Endpoint: "h:443",
	}})
	out := stripAnsi(ns.(*Model).View(80, 24))
	for _, want := range []string{"connected", "home", "sing-box", "h:443"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestOpDoneUpRefreshes(t *testing.T) {
	conn := &testutil.FakeConn{}
	st := New(context.Background(), conn, theme.Default(), false)
	ns, _ := st.Update(shared.StatusMsg{St: app.StatusView{Running: false, ProfileName: "home"}})
	ns2, _ := ns.Update(kf("enter")) // busy
	ns3, cmd := ns2.Update(shared.OpDoneMsg{Op: "up", Label: "home"})
	if cmd == nil {
		t.Fatal("must refetch status after up")
	}
	_ = ns3
	// error path
	ns4, _ := ns2.Update(shared.OpDoneMsg{Op: "up", Err: domain.ErrCoreNotFound})
	if out := stripAnsi(ns4.(*Model).View(80, 24)); !strings.Contains(out, "sing-box") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestNoProfileHint(t *testing.T) {
	conn := &testutil.FakeConn{}
	st := New(context.Background(), conn, theme.Default(), false)
	ns, _ := st.Update(shared.StatusMsg{St: app.StatusView{}})
	ns2, _ := ns.Update(kf("enter"))
	if len(conn.UpCalls) != 0 {
		t.Fatal("must not dial without profile")
	}
	out := stripAnsi(ns2.(*Model).View(80, 24))
	if !strings.Contains(out, "Profiles") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestRefreshSpinnerExpiry(t *testing.T) {
	conn := &testutil.FakeConn{}
	st := New(context.Background(), conn, theme.Default(), false)
	ns, _ := st.Update(shared.StatusMsg{St: app.StatusView{}})
	if _, cmd := ns.Update(kf("r")); cmd == nil {
		t.Fatal("r must refresh")
	}
	// busy spinner advances
	busy := ns.(*Model)
	busy.busy = true
	busy.busyOp = "up"
	if _, cmd := busy.Update(spinnerTick()); cmd == nil {
		t.Fatal("spinner must tick while busy")
	}
	// toast expiry hides
	ns2, _ := busy.Update(shared.OpDoneMsg{Op: "down"})
	if out := stripAnsi(ns2.(*Model).View(80, 24)); !strings.Contains(out, "Отключено") {
		t.Fatalf("got:\n%s", out)
	}
}

func spinnerTick() tea.Msg { return spinner.TickMsg{} }

var _ = time.Second

func TestStatusMsgErrorView(t *testing.T) {
	conn := &testutil.FakeConn{}
	st := New(context.Background(), conn, theme.Default(), false)
	ns, _ := st.Update(shared.StatusMsg{Err: domain.ErrCoreNotFound})
	out := stripAnsi(ns.(*Model).View(80, 24))
	if !strings.Contains(out, "sing-box") {
		t.Fatalf("got:\n%s", out)
	}
}
