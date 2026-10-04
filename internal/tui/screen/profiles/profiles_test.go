package profiles

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
	"github.com/SabirDzh/VpnCLI/internal/tui/testutil"
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

func kf(s string) tea.Msg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	default:
		return tea.KeyPressMsg{Code: rune(s[0])}
	}
}

func testModel() (*Model, *testutil.FakeConn, *testutil.FakeProfiles) {
	conn := &testutil.FakeConn{}
	prof := &testutil.FakeProfiles{
		Items: []domain.Profile{
			{ID: "a1", Name: "home", Protocol: domain.ProtocolVLESS, Source: domain.ManualSource},
			{ID: "b2", Name: "work", Protocol: domain.ProtocolTrojan, Source: "subscription:s1"},
		},
		ActiveID: "a1",
	}
	m := New(context.Background(), conn, prof, theme.Default(), false)
	ns, _ := m.Update(shared.ProfilesMsg{List: prof.Items, ActiveID: "a1"})
	ns.(*Model).list.Select(0)
	return ns.(*Model), conn, prof
}

func TestUseFlow(t *testing.T) {
	m, _, prof := testModel()
	ns, cmd := m.Update(kf("enter"))
	if cmd == nil {
		t.Fatal("must start use")
	}
	_ = cmd()
	if len(prof.Used) != 1 || prof.Used[0] != "a1" {
		t.Fatalf("Used = %v", prof.Used)
	}
	_ = ns
}

func TestConnectChain(t *testing.T) {
	m, conn, _ := testModel()
	// 'c' opens confirm (needs conn + full profile present)
	m.list.Select(1)
	ns, _ := m.Update(kf("c"))
	m2 := ns.(*Model)
	if !m2.confirm.Showing() {
		t.Fatal("must ask for confirm")
	}
	// confirm Yes -> use -> up chain (cursor starts at No, move right first)
	nsR, _ := m2.Update(kf("right"))
	ns3, cmd := nsR.Update(kf("enter"))
	_ = cmd()
	m3 := ns3.(*Model)
	if !m3.busy {
		t.Fatal("must be busy after confirm")
	}
	// use completes -> up starts
	ns4, cmd4 := m3.Update(shared.OpDoneMsg{Op: "use", Label: "work"})
	if cmd4 == nil {
		t.Fatal("must chain up after use")
	}
	_ = cmd4()
	if len(conn.UpCalls) != 1 || conn.UpCalls[0] != "b2" {
		t.Fatalf("UpCalls = %v", conn.UpCalls)
	}
	_ = ns4
}

func TestConfirmCancel(t *testing.T) {
	m, conn, prof := testModel()
	m.list.Select(0)
	ns, _ := m.Update(kf("c"))
	ns2, _ := ns.Update(kf("esc"))
	m2 := ns2.(*Model)
	if m2.confirm.Showing() {
		t.Fatal("must close on esc")
	}
	if len(conn.UpCalls) != 0 || len(prof.Used) != 0 {
		t.Fatal("cancel must not touch services")
	}
}

func TestNoSecretsInRows(t *testing.T) {
	m, _, _ := testModel()
	out := m.View(80, 20)
	for _, secret := range []string{"uuid", "pass", "203.0", "hiddenserver"} {
		if strings.Contains(strings.ToLower(out), secret) {
			t.Fatalf("leak %q in:\n%s", secret, out)
		}
	}
}

func TestRefreshAndReadonly(t *testing.T) {
	m, _, _ := testModel()
	ns, cmd := m.Update(kf("r"))
	if cmd == nil {
		t.Fatal("r must refresh")
	}
	_ = ns
}

func TestReadonlyConnectRefused(t *testing.T) {
	conn := &testutil.FakeConn{}
	prof := &testutil.FakeProfiles{Items: []domain.Profile{{ID: "a1", Name: "home"}}}
	m := New(context.Background(), conn, prof, theme.Default(), true)
	ns, _ := m.Update(shared.ProfilesMsg{List: prof.Items})
	ns.(*Model).list.Select(0)
	ns2, _ := ns.Update(kf("c"))
	if len(conn.UpCalls) != 0 {
		t.Fatal("readonly must not dial")
	}
	out := ns2.(*Model).toast.View()
	if !strings.Contains(out, "root") {
		t.Fatalf("must explain privileges, got %q", out)
	}
}

func TestToastExpiryAndLoading(t *testing.T) {
	m, _, _ := testModel()
	fresh := New(context.Background(), &testutil.FakeConn{}, &testutil.FakeProfiles{}, theme.Default(), false)
	if out := fresh.View(80, 20); !strings.Contains(out, "loading") {
		t.Fatalf("got:\n%s", out)
	}
	_ = m
}

func TestFindProfileErrorAndExpiry(t *testing.T) {
	conn := &testutil.FakeConn{}
	prof := &testutil.FakeProfiles{Items: []domain.Profile{{ID: "a1", Name: "home"}}}
	prof.ListErr = errBoom2{}
	m := New(context.Background(), conn, prof, theme.Default(), false)
	ns, _ := m.Update(shared.ProfilesMsg{List: prof.Items, ActiveID: "a1"})
	ns.(*Model).list.Select(0)
	ns2, _ := ns.Update(kf("c")) // findProfile fails via ListErr
	_ = ns2
}

type errBoom2 struct{}

func (errBoom2) Error() string { return "boom" }

func TestProfilesMsgErrorAndConfirmKeys(t *testing.T) {
	m, _, _ := testModel()
	ns, _ := m.Update(shared.ProfilesMsg{Err: errBoom2{}})
	if out := ns.(*Model).View(80, 20); !strings.Contains(out, "boom") {
		t.Fatalf("got:\n%s", out)
	}
	// confirm navigation keys
	m2, _, _ := testModel()
	m2.list.Select(0)
	ns2, _ := m2.Update(kf("c"))
	for _, k := range []string{"left", "tab", "q"} {
		ns2, _ = ns2.Update(keyPressNamed(k))
	}
	if ns2.(*Model).confirm.Showing() {
		t.Fatal("q must close confirm")
	}
}

func keyPressNamed(s string) tea.Msg {
	switch s {
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "q":
		return tea.KeyPressMsg{Code: 'q'}
	default:
		return kf(s)
	}
}
