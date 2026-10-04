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
	return ns.(*Model), conn, prof
}

func sizedView(m *Model, w, h int) string {
	ns, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return ns.(*Model).View(w, h)
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

func TestNavigation(t *testing.T) {
	m, _, _ := testModel()
	ns, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	if ns.(*Model).cursor != 1 {
		t.Fatal("j must move down")
	}
	ns, _ = ns.Update(tea.KeyPressMsg{Code: 'k'})
	if ns.(*Model).cursor != 0 {
		t.Fatal("k must move up")
	}
}

func TestConnectChain(t *testing.T) {
	m, conn, _ := testModel()
	m.cursor = 1
	ns, _ := m.Update(kf("c"))
	m2 := ns.(*Model)
	if !m2.confirm.Showing() {
		t.Fatal("must ask for confirm")
	}
	// confirm defaults to Yes -> use -> up chain
	ns3, cmd := m2.Update(kf("enter"))
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
	out := sizedView(m, 80, 20)
	for _, secret := range []string{"uuid", "pass", "203.0", "hiddenserver"} {
		if strings.Contains(strings.ToLower(out), secret) {
			t.Fatalf("leak %q in:\n%s", secret, out)
		}
	}
}

func TestFilter(t *testing.T) {
	m, _, _ := testModel()
	ns, _ := m.Update(kf("/"))
	mm := ns.(*Model)
	if !mm.filtering {
		t.Fatal("must enter filter mode")
	}
	ns, _ = mm.Update(tea.KeyPressMsg{Code: 'w'})
	mm = ns.(*Model)
	if out := mm.View(80, 20); !strings.Contains(out, "work") || strings.Contains(out, "home\n") {
		t.Fatalf("filter must narrow rows:\n%s", out)
	}
	// esc clears the filter
	ns, _ = mm.Update(kf("esc"))
	mm = ns.(*Model)
	if mm.filtering || mm.filter != "" {
		t.Fatal("esc must clear filter")
	}
	if out := mm.View(80, 20); !strings.Contains(out, "home") {
		t.Fatal("rows must return")
	}
}

func TestHeaderActiveAndEmpty(t *testing.T) {
	m, _, _ := testModel()
	out := sizedView(m, 80, 20)
	if !strings.Contains(out, "● home") {
		t.Fatalf("must show active name:\n%s", out)
	}
	empty := New(context.Background(), &testutil.FakeConn{}, &testutil.FakeProfiles{}, theme.Default(), false)
	ns, _ := empty.Update(shared.ProfilesMsg{})
	if out := ns.(*Model).View(80, 20); !strings.Contains(out, "vpn profile add") {
		t.Fatalf("must hint add:\n%s", out)
	}
}

func TestFindProfileErrorAndExpiry(t *testing.T) {
	conn := &testutil.FakeConn{}
	prof := &testutil.FakeProfiles{Items: []domain.Profile{{ID: "a1", Name: "home"}}}
	prof.ListErr = errBoom2{}
	m := New(context.Background(), conn, prof, theme.Default(), false)
	ns, _ := m.Update(shared.ProfilesMsg{List: prof.Items, ActiveID: "a1"})
	ns2, _ := ns.Update(kf("c")) // findProfile fails via ListErr
	_ = ns2
}

type errBoom2 struct{}

func (errBoom2) Error() string { return "boom" }

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
	ns2, _ := ns.Update(kf("c"))
	if len(conn.UpCalls) != 0 {
		t.Fatal("readonly must not dial")
	}
	out := ns2.(*Model).toast.View()
	if !strings.Contains(out, "root") {
		t.Fatalf("must explain privileges, got %q", out)
	}
}

func TestBadgesRendered(t *testing.T) {
	m, _, _ := testModel()
	ns, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	out := ns.(*Model).View(80, 20)
	for _, want := range []string{"[VLESS]", "[TROJAN]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("must render badge %s in:\n%s", want, out)
		}
	}
}

func typeText(m *Model, s string) *Model {
	for _, r := range s {
		ns, _ := m.Update(tea.KeyPressMsg{Code: r})
		m = ns.(*Model)
	}
	return m
}

func TestAddProfileFlow(t *testing.T) {
	m, _, prof := testModel()
	ns, _ := m.Update(kf("a"))
	mm := ns.(*Model)
	if !mm.input.Showing() {
		t.Fatal("a must open input")
	}
	mm = typeText(mm, "trojan://pw@h:1#n")
	ns, cmd := mm.Update(kf("enter"))
	if cmd == nil {
		t.Fatal("enter must submit")
	}
	_ = cmd()
	if len(prof.Added) != 1 {
		t.Fatalf("Added = %v", prof.Added)
	}
	ns, _ = ns.Update(shared.OpDoneMsg{Op: "add", Label: "n"})
	_ = ns
}

func TestDeleteProfileFlow(t *testing.T) {
	m, _, prof := testModel()
	prof.Items = append(prof.Items, domain.Profile{ID: "z9", Name: "gone"})
	ns, _ := m.Update(shared.ProfilesMsg{List: prof.Items, ActiveID: "a1"})
	mm := ns.(*Model)
	mm.cursor = 2
	ns, _ = mm.Update(kf("x"))
	mm = ns.(*Model)
	if !mm.confirm.Showing() {
		t.Fatal("x must ask confirm")
	}
	ns, cmd := mm.Update(kf("enter"))
	_ = cmd()
	if len(prof.Removed) != 1 || prof.Removed[0] != "z9" {
		t.Fatalf("Removed = %v", prof.Removed)
	}
	_ = ns
}

func TestEditProfileFlow(t *testing.T) {
	m, _, prof := testModel()
	ns, _ := m.Update(kf("e"))
	mm := ns.(*Model)
	if !mm.input.Showing() {
		t.Fatal("e must open edit form")
	}
	if got := mm.input.Value(); got != "home" {
		t.Fatalf("name prefilled = %q", got)
	}
	mm = typeText(mm, "X")
	ns, _ = mm.Update(kf("enter"))
	mm = ns.(*Model)
	if !mm.input.Showing() {
		t.Fatal("must advance to uri step")
	}
	ns, cmd := mm.Update(kf("enter"))
	if cmd == nil {
		t.Fatal("enter must submit")
	}
	_ = cmd()
	if len(prof.Edited) != 1 {
		t.Fatalf("Edited = %+v", prof.Edited)
	}
	e := prof.Edited[0]
	if e.ID != "a1" || e.Name != "homeX" || e.Value != "" {
		t.Fatalf("Edited = %+v", prof.Edited)
	}
	ns, _ = ns.Update(shared.OpDoneMsg{Op: "edit", Label: "homeX"})
	if out := ns.(*Model).toast.View(); !strings.Contains(out, "Изменён") {
		t.Fatalf("toast = %q", out)
	}
}

func TestEditSubscriptionOwnedRefused(t *testing.T) {
	m, _, _ := testModel()
	m.cursor = 1 // "work" comes from a subscription
	ns, _ := m.Update(kf("e"))
	mm := ns.(*Model)
	if mm.input.Showing() {
		t.Fatal("must not edit subscription-owned profile")
	}
	if out := mm.toast.View(); !strings.Contains(out, "подписк") {
		t.Fatalf("toast = %q", out)
	}
}

func TestConfirmFlushLeft(t *testing.T) {
	m, _, _ := testModel()
	ns, _ := m.Update(kf("x"))
	out := ns.(*Model).View(80, 24)
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "╭") {
			return
		}
	}
	t.Fatalf("confirm box must start at column 0:\n%s", out)
}
