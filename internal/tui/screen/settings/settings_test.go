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
		LogLevel: "info", TUNEnabled: true, MTU: 9000, Stack: "system",
		AutoRoute: true, StrictRoute: true, MixedPort: 10808,
		Privileged: false, ConfigDir: "/c", DataDir: "/d",
		StateFile: "/s", LogFile: "/l",
		SplitExclude:     []string{"bank.example", "192.168.0.0/16"},
		SplitMode:        "exclude",
		PresetApps:       true,
		Multiplex:        "auto",
		DNSServers:       []string{"1.1.1.1", "8.8.8.8"},
		DNSStrategy:      "prefer_ipv4",
		SplitExcludeApps: []string{"Госуслуги"},
	}
}

// downToCat2 moves to a row with the given kind and cat.
func downToCat2(t *testing.T, m *Model, k rowKind, cat string) *Model {
	t.Helper()
	return downToLabel(t, m, func(r row) bool { return r.kind == k && r.cat == cat })
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
	return downToLabel(t, m, func(r row) bool { return r.kind == k })
}

// downToCat moves the cursor until it rests on a blocklist row of the
// given category.
func downToCat(t *testing.T, m *Model, cat string) *Model {
	t.Helper()
	return downToLabel(t, m, func(r row) bool { return r.kind == kindBlocklist && r.cat == cat })
}

func downToLabel(t *testing.T, m *Model, match func(row) bool) *Model {
	t.Helper()
	for i := 0; i < 40; i++ {
		if match(m.rows[m.cursor]) {
			return m
		}
		ns, _ := m.Update(kf('j'))
		m = ns.(*Model)
	}
	t.Fatal("target row not reached")
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
	for _, want := range []string{"adblock", "trackerblock", "socialblock", "kill switch", "app firewall", "split exclude", "split include"} {
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
	m = downToCat(t, m, "ads")
	ns, cmd := m.Update(tea.KeyPressMsg{Code: ' '})
	mm := ns.(*Model)
	_ = cmd
	if len(api.BlockCalls) != 1 || api.BlockCalls[0] != (fakes.BlockCall{Kind: "ads", On: true}) {
		t.Fatalf("BlockCalls = %v", api.BlockCalls)
	}
	if out := mm.View(80, 40); !strings.Contains(out, "Сохранено") {
		t.Fatalf("must toast save, got:\n%s", out)
	}
	// snapshot refreshed: fake flips Info.Adblock, view must show "on"
	if !strings.Contains(mm.View(80, 40), "on") {
		t.Fatal("adblock must render on after toggle")
	}
}

func TestToggleSocialBlock(t *testing.T) {
	m, api := newModel(nil)
	m = downToCat(t, m, "social")
	ns, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	_ = ns
	if len(api.BlockCalls) != 1 || api.BlockCalls[0] != (fakes.BlockCall{Kind: "social", On: true}) {
		t.Fatalf("BlockCalls = %v", api.BlockCalls)
	}
}

func TestSplitModeCycle(t *testing.T) {
	m, api := newModel(nil)
	m = downToCat2(t, m, kindCycle, "mode")
	ns, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	_ = ns
	if len(api.SplitModeCalls) != 1 || api.SplitModeCalls[0] != "include" {
		t.Fatalf("mode calls: %v", api.SplitModeCalls)
	}
}

func TestToggleTUNAndPreset(t *testing.T) {
	m, api := newModel(nil)
	m = downToCat2(t, m, kindToggle, "tun")
	ns, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	_ = ns
	if len(api.TUNCalls) != 1 || api.TUNCalls[0] != false {
		t.Fatalf("tun calls: %v", api.TUNCalls)
	}
	m = downToCat2(t, m, kindToggle, "preset")
	ns, _ = m.Update(tea.KeyPressMsg{Code: ' '})
	_ = ns
	if len(api.PresetCalls) != 1 || api.PresetCalls[0] != false {
		t.Fatalf("preset calls: %v", api.PresetCalls)
	}
}

func TestEditSplitAppsExclude(t *testing.T) {
	m, api := newModel(nil)
	m = downTo(t, m, kindAppExclude)
	ns, _ := m.Update(kf('\r'))
	mm := ns.(*Model)
	if !mm.input.Showing() {
		t.Fatal("must open input")
	}
	if got := mm.input.Value(); got != "Госуслуги" {
		t.Fatalf("prefilled %q", got)
	}
	ns, _ = mm.Update(kf('X'))
	_, cmd := ns.Update(kf('\r'))
	if cmd == nil {
		t.Fatal("enter must submit")
	}
	_ = cmd()
	if len(api.SplitAppsCalls) != 1 || api.SplitAppsCalls[0].Which != "exclude" ||
		len(api.SplitAppsCalls[0].Apps) != 1 || api.SplitAppsCalls[0].Apps[0] != "ГосуслугиX" {
		t.Fatalf("calls: %+v", api.SplitAppsCalls)
	}
}

func TestEditDNSServers(t *testing.T) {
	m, api := newModel(nil)
	m = downTo(t, m, kindDNS)
	ns, _ := m.Update(kf('\r'))
	mm := ns.(*Model)
	if got := mm.input.Value(); got != "1.1.1.1, 8.8.8.8" {
		t.Fatalf("prefilled %q", got)
	}
	ns, _ = mm.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	ns, _ = ns.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	ns, _ = ns.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	ns, _ = ns.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	ns, _ = ns.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	ns, _ = ns.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	ns, _ = ns.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	ns, _ = ns.Update(kf('9'))
	ns, _ = ns.Update(kf('.'))
	ns, _ = ns.Update(kf('9'))
	ns, _ = ns.Update(kf('.'))
	ns, _ = ns.Update(kf('9'))
	ns, _ = ns.Update(kf('.'))
	ns, _ = ns.Update(kf('9'))
	_, cmd := ns.Update(kf('\r'))
	_ = cmd()
	if len(api.DNSServersCalls) != 1 {
		t.Fatalf("dns calls: %v", api.DNSServersCalls)
	}
	if api.DNSServersCalls[0][1] != "9.9.9.9" {
		t.Fatalf("servers: %v", api.DNSServersCalls[0])
	}
}

func TestEditMTUNum(t *testing.T) {
	m, api := newModel(nil)
	m = downToCat2(t, m, kindNum, "mtu")
	ns, _ := m.Update(kf('\r'))
	mm := ns.(*Model)
	if got := mm.input.Value(); got != "9000" {
		t.Fatalf("prefilled %q", got)
	}
	// backspace twice → "90", append "00" → 9000 again; then 5 → 90005 invalid
	ns, _ = mm.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	ns, _ = ns.Update(kf('1'))
	_, cmd := ns.Update(kf('\r'))
	_ = cmd()
	if len(api.MTUCalls) != 1 || api.MTUCalls[0] != 9001 {
		t.Fatalf("mtu calls: %v", api.MTUCalls)
	}
}

func TestEditMTUInvalidKeepsInput(t *testing.T) {
	m, api := newModel(nil)
	m = downToCat2(t, m, kindNum, "mtu")
	ns, _ := m.Update(kf('\r'))
	mm := ns.(*Model)
	ns, _ = mm.Update(kf('a'))
	ns, cmd := ns.Update(kf('\r'))
	mm = ns.(*Model)
	if cmd == nil {
		t.Fatal("enter must be handled")
	}
	_ = cmd()
	if len(api.MTUCalls) != 0 {
		t.Fatalf("invalid input must not save: %v", api.MTUCalls)
	}
	if !mm.input.Showing() {
		t.Fatal("input must stay open on invalid number")
	}
}

func TestToggleKillSwitch(t *testing.T) {
	m, api := newModel(nil)
	m = downTo(t, m, kindKillSwitch)
	if got := m.rows[m.cursor].value; got != "off" {
		t.Fatalf("kill switch must start off, got %q", got)
	}
	ns, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	mm := ns.(*Model)
	if len(api.KillSwitchCalls) != 1 || !api.KillSwitchCalls[0] {
		t.Fatalf("KillSwitchCalls = %v", api.KillSwitchCalls)
	}
	if got := mm.rows[m.cursor].value; got != "on" {
		t.Fatalf("kill switch must render on, got %q", got)
	}
}

func TestEditAppFirewall(t *testing.T) {
	m, api := newModel(nil)
	m = downTo(t, m, kindAppFirewall)
	ns, _ := m.Update(kf('\r'))
	mm := ns.(*Model)
	if !mm.input.Showing() {
		t.Fatal("enter on app firewall row must open input")
	}
	ns, _ = mm.Update(kf('t'))
	ns, _ = ns.Update(kf('o'))
	_, cmd := ns.Update(kf('\r'))
	if cmd == nil {
		t.Fatal("enter must submit")
	}
	_ = cmd()
	if len(api.AppFirewallCalls) != 1 || len(api.AppFirewallCalls[0]) != 1 || api.AppFirewallCalls[0][0] != "to" {
		t.Fatalf("AppFirewallCalls = %v", api.AppFirewallCalls)
	}
}

func TestToggleErrorToast(t *testing.T) {
	api := &fakes.FakeSettings{Info: testInfo(), SetErr: errBoom{}}
	m, _ := newModel(api)
	m = downToCat(t, m, "ads")
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
