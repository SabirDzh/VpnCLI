package settings

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/tui/fakes"
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

func testInfo() shared.SettingsInfo {
	return shared.SettingsInfo{
		CoreDefault: "sing-box", SingBoxVersion: "1.14.2", MinVersion: "1.14.0",
		LogLevel: "info", TUNEnabled: true, MTU: 9000,
		AutoRoute: true, StrictRoute: true, MixedPort: 10808,
		Privileged: false, ConfigDir: "/c", DataDir: "/d",
		StateFile: "/s", LogFile: "/l",
		SplitExclude: []string{"bank.example", "192.168.0.0/16"},
	}
}

func newModel(api *fakes.FakeSettings) (*Model, *fakes.FakeSettings) {
	if api == nil {
		api = &fakes.FakeSettings{Info: testInfo()}
	}
	return New(api, api.Info, theme.Default()), api
}

func kf(c byte) tea.Msg { return tea.KeyPressMsg{Code: rune(c)} }

// downTo moves the cursor until it rests on a row of the given kind.
func downTo(t *testing.T, m *Model, k rowKind) *Model {
	t.Helper()
	for i := 0; i < 40; i++ {
		if m.rows[m.cursor].kind == k {
			return m
		}
		ns, _ := m.Update(kf('j'))
		m = ns.(*Model)
	}
	t.Fatalf("row kind %d not reached", k)
	return m
}

func TestRendersSections(t *testing.T) {
	m, _ := newModel(nil)
	out := m.View(80, 40)
	for _, want := range []string{"Core", "Network", "App", "1.14.2", "9000", "no — TUN needs sudo"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestRendersFeatureRows(t *testing.T) {
	m, _ := newModel(nil)
	out := m.View(80, 40)
	for _, want := range []string{"adblock", "trackerblock", "split exclude", "split include"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "bank.example") || !strings.Contains(out, "2") {
		t.Fatalf("exclude list must render:\n%s", out)
	}
}

func TestCoreError(t *testing.T) {
	api := &fakes.FakeSettings{Info: testInfo()}
	api.Info.SingBoxVersion = ""
	api.Info.SingBoxErr = "not found in PATH"
	m := New(api, api.Info, theme.Default())
	if out := m.View(80, 40); !strings.Contains(out, "not found in PATH") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestToggleAdblock(t *testing.T) {
	m, api := newModel(nil)
	m = downTo(t, m, kindAdblock)
	ns, cmd := m.Update(tea.KeyPressMsg{Code: ' '})
	mm := ns.(*Model)
	_ = cmd
	if len(api.AdblockCalls) != 1 || !api.AdblockCalls[0] {
		t.Fatalf("AdblockCalls = %v", api.AdblockCalls)
	}
	if out := mm.View(80, 40); !strings.Contains(out, "Сохранено") {
		t.Fatalf("must toast save, got:\n%s", out)
	}
	// snapshot refreshed: fake flips Info.Adblock, view must show "on"
	if !strings.Contains(mm.View(80, 40), "on") {
		t.Fatal("adblock must render on after toggle")
	}
}

func TestToggleErrorToast(t *testing.T) {
	api := &fakes.FakeSettings{Info: testInfo(), SetErr: errBoom{}}
	m, _ := newModel(api)
	m = downTo(t, m, kindAdblock)
	ns, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	if out := ns.(*Model).View(80, 40); !strings.Contains(out, "Ошибка") {
		t.Fatalf("must toast error:\n%s", out)
	}
}

func TestEditSplitExclude(t *testing.T) {
	m, api := newModel(nil)
	m = downTo(t, m, kindExclude)
	ns, _ := m.Update(kf('\r'))
	mm := ns.(*Model)
	if !mm.input.Showing() {
		t.Fatal("enter on split row must open input")
	}
	if got := mm.input.Value(); got != "bank.example, 192.168.0.0/16" {
		t.Fatalf("prefilled = %q", got)
	}
	// edit: append to the first entry, submit
	ns, _ = mm.Update(kf('X'))
	_, cmd := ns.Update(kf('\r'))
	if cmd == nil {
		t.Fatal("enter must submit")
	}
	_ = cmd()
	if len(api.SplitCalls) != 1 {
		t.Fatalf("SplitCalls = %v", api.SplitCalls)
	}
	ex, inc := api.SplitCalls[0][0], api.SplitCalls[0][1]
	if len(ex) != 2 || ex[0] != "bank.example" || ex[1] != "192.168.0.0/16X" {
		t.Fatalf("exclude = %v", ex)
	}
	if len(inc) != 0 {
		t.Fatalf("include must be untouched: %v", inc)
	}
}

func TestEditSplitCancel(t *testing.T) {
	m, api := newModel(nil)
	m = downTo(t, m, kindExclude)
	ns, _ := m.Update(kf('\r'))
	ns, _ = ns.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	mm := ns.(*Model)
	if mm.input.Showing() {
		t.Fatal("esc must close input")
	}
	if len(api.SplitCalls) != 0 {
		t.Fatal("esc must not save")
	}
}

func TestRefreshSnapshot(t *testing.T) {
	m, api := newModel(nil)
	before := api.Snapshots
	ns, cmd := m.Update(kf('r'))
	_ = ns
	if cmd != nil {
		_ = cmd()
	}
	if api.Snapshots <= before {
		t.Fatal("r must re-read snapshot")
	}
}

func TestScrollAndBack(t *testing.T) {
	m, _ := newModel(nil)
	ns, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	mm := ns.(*Model)
	if mm.width != 80 {
		t.Fatal("must store width")
	}
	ns, cmd := mm.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	_ = ns
	if cmd == nil {
		t.Fatal("esc must go back")
	}
	if _, ok := cmd().(shared.BackMsg); !ok {
		t.Fatal("esc must produce BackMsg")
	}
	if m.Title() != "Settings" {
		t.Fatal("title")
	}
	if len(m.Keys()) == 0 {
		t.Fatal("keys must be advertised")
	}
	if m.Init() != nil {
		t.Fatal("no init fetch")
	}
}

func TestScrollOverflowNoPanic(t *testing.T) {
	m, _ := newModel(nil)
	down := tea.KeyPressMsg{Code: 'j'}
	ns := shared.Screen(m)
	for i := 0; i < 100; i++ {
		ns, _ = ns.Update(down)
	}
	_ = ns.View(80, 5)
	up := tea.KeyPressMsg{Code: 'k'}
	for i := 0; i < 100; i++ {
		ns, _ = ns.Update(up)
	}
	_ = ns.View(80, 30)
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }
