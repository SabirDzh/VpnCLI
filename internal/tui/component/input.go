package component

import (
	"github.com/SabirDzh/VpnCLI/internal/tui/theme"
)

// Input is a single-line text field for modal forms.
type Input struct {
	title  string
	value  []rune
	cursor int
	show   bool
	styles theme.Styles
}

// NewInput creates a hidden field.
func NewInput(st theme.Styles) Input { return Input{styles: st} }

// Open shows the field with a title and optional initial value.
func (in *Input) Open(title, initial string) {
	in.title = title
	in.value = []rune(initial)
	in.cursor = len(in.value)
	in.show = true
}

// Showing reports whether the field is on screen.
func (in Input) Showing() bool { return in.show }

// Close hides the field.
func (in *Input) Close() { in.show = false }

// Value returns the current text.
func (in Input) Value() string { return string(in.value) }

// Key handles editing keys. Returns handled=true when consumed.
func (in *Input) Key(k string) bool {
	if !in.show {
		return false
	}
	switch k {
	case "backspace":
		if in.cursor > 0 {
			in.value = append(in.value[:in.cursor-1], in.value[in.cursor:]...)
			in.cursor--
		}
		return true
	case "left":
		if in.cursor > 0 {
			in.cursor--
		}
		return true
	case "right":
		if in.cursor < len(in.value) {
			in.cursor++
		}
		return true
	case "space":
		in.insert(' ')
		return true
	default:
		if len(k) == 1 {
			in.insert(rune(k[0]))
			return true
		}
	}
	return false
}

func (in *Input) insert(r rune) {
	in.value = append(in.value[:in.cursor], append([]rune{r}, in.value[in.cursor:]...)...)
	in.cursor++
}

// View renders the field box.
func (in Input) View() string {
	if !in.show {
		return ""
	}
	before, after := string(in.value[:in.cursor]), string(in.value[in.cursor:])
	line := before + "▌" + after
	if line == "▌" {
		line = in.styles.Dim.Render("▌")
	}
	body := in.title + "\n\n" + line + "\n\n" + in.styles.Dim.Render("enter — ок · esc — отмена")
	return in.styles.Box.Render(body)
}
