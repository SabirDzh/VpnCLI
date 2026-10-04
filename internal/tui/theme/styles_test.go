package theme

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
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

// TestBoldTypography pins the heavier type scale: menu rows, selected
// rows, section headers and field labels render bold.
func TestBoldTypography(t *testing.T) {
	d := Default()
	for name, style := range map[string]lipgloss.Style{
		"SelectedRow": d.SelectedRow,
		"Label":       d.Label,
		"FieldLabel":  d.FieldLabel,
		"Section":     d.Section,
		"MenuTitle":   d.MenuTitle,
		"MenuSummary": d.MenuSummary,
	} {
		out := style.Render("x")
		// bold SGR may be merged with color params: \x1b[1m, \x1b[1;90m, \x1b[90;1m
		bold := strings.Contains(out, "\x1b[1m") || strings.Contains(out, "[1;") ||
			strings.Contains(out, ";1;") || strings.Contains(out, ";1m")
		if !bold {
			t.Fatalf("%s must render bold, got %q", name, out)
		}
	}
}
