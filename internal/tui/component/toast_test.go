package component

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

type expireMsg struct{ id int }

func expire(id int) tea.Msg { return expireMsg{id: id} }

func TestToastLifecycle(t *testing.T) {
	to := NewToast(theme.Default())
	if to.Visible() {
		t.Fatal("must start hidden")
	}
	if to.View() != "" {
		t.Fatal("hidden toast renders nothing")
	}
	if cmd := to.Show("hello", true, expire); cmd == nil {
		t.Fatal("must schedule expiry")
	}
	if !to.Visible() {
		t.Fatal("must be visible after Show")
	}
	if !strings.Contains(to.View(), "hello") {
		t.Fatal("must render text")
	}
	to.Expire(9999) // stale id ignored
	if !to.Visible() {
		t.Fatal("stale expiry must be ignored")
	}
	to.Expire(1) // first Show assigns id 1
	if to.Visible() {
		t.Fatal("must hide after expiry")
	}
	_ = to.Show("bad", false, expire)
	if !strings.Contains(to.View(), "bad") {
		t.Fatal("must render errors")
	}
}

func TestConfirm(t *testing.T) {
	c := NewConfirm(theme.Default())
	if c.Showing() {
		t.Fatal("must start hidden")
	}
	c.Ask("Удалить?", "delete")
	if !c.Showing() || c.Tag() != "delete" {
		t.Fatal("must show with tag")
	}
	if !strings.Contains(c.View(80), "Удалить?") {
		t.Fatal("must render title")
	}
	c.Move() // cursor -> Yes
	tag, ok := c.Resolve()
	if tag != "delete" || !ok {
		t.Fatal("must resolve Yes with tag")
	}
	if c.Showing() {
		t.Fatal("must hide after resolve")
	}
	c.Ask("Ещё?", "x") // cursor defaults to No
	_, ok = c.Resolve()
	if ok {
		t.Fatal("must resolve No")
	}
}
