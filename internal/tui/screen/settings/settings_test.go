package settings

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

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
	}
}

func TestRendersSections(t *testing.T) {
	m := New(testInfo(), theme.Default())
	out := m.View(80, 24)
	for _, want := range []string{"Core", "Network", "App", "1.14.2", "9000", "no — TUN needs sudo"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestCoreError(t *testing.T) {
	info := testInfo()
	info.SingBoxVersion = ""
	info.SingBoxErr = "not found in PATH"
	m := New(info, theme.Default())
	if out := m.View(80, 24); !strings.Contains(out, "not found in PATH") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestScrollAndBack(t *testing.T) {
	m := New(testInfo(), theme.Default())
	ns, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	mm := ns.(*Model)
	if mm.width != 80 {
		t.Fatal("must store width")
	}
	ns, _ = mm.Update(tea.KeyPressMsg{Code: 'j'})
	if ns.(*Model).offset != 1 {
		t.Fatal("j must scroll")
	}
	ns, _ = ns.Update(tea.KeyPressMsg{Code: 'k'})
	if ns.(*Model).offset != 0 {
		t.Fatal("k must scroll back")
	}
	ns, cmd := ns.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
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
	if m.Keys() != nil {
		t.Fatal("no own keys")
	}
	if m.Init() != nil {
		t.Fatal("no init fetch")
	}
}
