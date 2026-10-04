// Package other implements the "Other" tab: about info, update checks
// and installs, auto-update toggle and the repository link.
package other

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/app"
	"github.com/SabirDzh/VpnCLI/internal/tui/fakes"
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
	"github.com/SabirDzh/VpnCLI/internal/tui/testutil"
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

func testDeps() (*testutil.FakeUpdate, *fakes.FakeSettings, shared.Deps) {
	upd := &testutil.FakeUpdate{}
	st := &fakes.FakeSettings{Info: shared.SettingsInfo{}}
	deps := shared.Deps{
		Update:      upd,
		SettingsAPI: st,
		Version:     "v1.2.3",
		Repo:        "SabirDzh/VpnCLI",
	}
	return upd, st, deps
}

func kf(c byte) tea.Msg { return tea.KeyPressMsg{Code: rune(c)} }

// runCmd executes a command and every nested batch command, returning
// the produced message for dispatch back into Update.
func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				_ = c()
			}
		}
		return nil
	}
	return msg
}

func TestRendersAbout(t *testing.T) {
	_, _, deps := testDeps()
	m := New(deps, theme.Default())
	out := m.View(80, 24)
	for _, want := range []string{"v1.2.3", "SabirDzh/VpnCLI", "auto-updates", "off", "never"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestCheckUpToDate(t *testing.T) {
	upd, _, deps := testDeps()
	upd.Res = app.CheckResult{Current: "v1.2.3", Latest: "v1.2.3", Available: false}
	m := New(deps, theme.Default())
	ns, cmd := m.Update(kf('c'))
	ns, _ = ns.Update(runCmd(cmd))
	out := ns.(*Model).View(80, 24)
	if upd.CheckCalls != 1 {
		t.Fatal("c must check")
	}
	if !strings.Contains(out, "v1.2.3") || !strings.Contains(out, "Актуально") {
		t.Fatalf("must show up-to-date verdict:\n%s", out)
	}
}

func TestCheckAvailableAndInstall(t *testing.T) {
	upd, _, deps := testDeps()
	upd.Res = app.CheckResult{Current: "v1.2.3", Latest: "v1.3.0", Available: true}
	m := New(deps, theme.Default())
	ns, cmd := m.Update(kf('c'))
	ns, _ = ns.Update(runCmd(cmd))
	if out := ns.(*Model).View(80, 24); !strings.Contains(out, "v1.3.0") {
		t.Fatalf("must show newer version:\n%s", out)
	}
	ns, cmd = ns.Update(kf('U'))
	ns, _ = ns.Update(runCmd(cmd))
	if len(upd.UpdateCalls) != 1 || upd.UpdateCalls[0] != "v1.3.0" {
		t.Fatalf("UpdateCalls = %v", upd.UpdateCalls)
	}
	if out := ns.(*Model).View(80, 24); !strings.Contains(out, "перезапусти") {
		t.Fatalf("must ask to restart:\n%s", out)
	}
}

func TestUpdateInstallError(t *testing.T) {
	upd, _, deps := testDeps()
	upd.Res = app.CheckResult{Current: "v1.2.3", Latest: "v1.3.0", Available: true}
	upd.UpdateErr = errFail{}
	m := New(deps, theme.Default())
	ns, cmd := m.Update(kf('c'))
	ns, _ = ns.Update(runCmd(cmd))
	ns, cmd = ns.Update(kf('U'))
	ns, _ = ns.Update(runCmd(cmd))
	if out := ns.(*Model).View(80, 24); !strings.Contains(out, "fail") {
		t.Fatalf("must show install error:\n%s", out)
	}
}

func TestToggleAutoUpdate(t *testing.T) {
	upd, st, deps := testDeps()
	_ = upd
	m := New(deps, theme.Default())
	ns, _ := m.Update(kf('a'))
	mm := ns.(*Model)
	if len(st.AutoCalls) != 1 || !st.AutoCalls[0] {
		t.Fatalf("AutoCalls = %v", st.AutoCalls)
	}
	if !mm.auto {
		t.Fatal("auto must flip on")
	}
	if out := mm.View(80, 24); !strings.Contains(out, "auto-updates") || !strings.Contains(out, "on") || strings.Contains(out, "off") {
		t.Fatalf("must reflect new state:\n%s", out)
	}
}

func TestOpenRepo(t *testing.T) {
	_, _, deps := testDeps()
	m := New(deps, theme.Default())
	var opened string
	m.openURL = func(u string) error { opened = u; return nil }
	ns, _ := m.Update(kf('o'))
	_ = ns
	if opened != "https://github.com/SabirDzh/VpnCLI" {
		t.Fatalf("opened = %q", opened)
	}
}

func TestDatesRender(t *testing.T) {
	upd, _, deps := testDeps()
	now := time.Now()
	upd.CheckTime = now.Add(-time.Hour)
	upd.UpdatedTime = now.Add(-26 * time.Hour)
	m := New(deps, theme.Default())
	out := m.View(80, 24)
	if !strings.Contains(out, now.Add(-time.Hour).Format("02.01 15:04")) {
		t.Fatalf("last check date missing:\n%s", out)
	}
	if !strings.Contains(out, now.Add(-26*time.Hour).Format("02.01 15:04")) {
		t.Fatalf("last update date missing:\n%s", out)
	}
}

type errFail struct{}

func (errFail) Error() string { return "fail" }
