package theme

import (
	"strings"
	"testing"
)

// TestSelectedRowHasNoBackground pins the "selected = other text color"
// convention: the style must not paint a background bar.
func TestSelectedRowHasNoBackground(t *testing.T) {
	out := Default().SelectedRow.Render("x")
	for _, bg := range []string{"100m", "101m", "102m", "103m", "104m", "105m", "106m", "107m", "48;"} {
		if strings.Contains(out, bg) {
			t.Fatalf("SelectedRow must not set a background, got %q", out)
		}
	}
	if !strings.HasPrefix(out, "\x1b[") {
		t.Fatalf("SelectedRow must set a color, got %q", out)
	}
}
