package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// The failure mode this catches is a palette that renders ink on ink: a
// theme whose foreground and background land on the same colour for
// something the user actually has to read.
func TestThemesDontRenderInkOnInk(t *testing.T) {
	defer applyTheme("screen") // leave global state as every other test expects it

	for _, name := range themeOrder {
		applyTheme(name)

		styles := map[string]lipgloss.Style{
			"status bar":  noteStyle,
			"help chip":   helpChipStyle,
			"cursor":      cursorStyle,
			"overlay":     overlayStyle,
			"overlay dim": overlayDimStyle,
			"select":      selectStyle,
		}
		for label, s := range styles {
			if fg, bg := s.GetForeground(), s.GetBackground(); fg == bg {
				t.Errorf("theme %q: %s renders %v on itself", name, label, fg)
			}
		}
	}
}
