package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// Every overlay has to come out as a rectangle that fits on the screen. A line
// one column too long wraps, and a wrapped line shifts everything below it —
// which is how a panel turns into a smear halfway down the page.
func TestPanelsAreRectanglesThatFit(t *testing.T) {
	sizes := []struct{ w, h int }{
		{200, 60}, // a wide monitor
		{80, 24},  // the usual
		{60, 18},  // a tmux split
		{34, 12},  // as small as the editor still draws
	}
	panels := map[string]Mode{
		"help":    ModeHelp,
		"about":   ModeAbout,
		"stats":   ModeStats,
		"confirm": ModeConfirm,
		"open":    ModeOpen,
		"save as": ModeSaveAs,
	}

	for name, mode := range panels {
		for _, size := range sizes {
			t.Run(name, func(t *testing.T) {
				a := testApp(t)
				a.w, a.h = size.w, size.h
				a.mode = mode
				a.confirm = confirmation{message: "discard changes?"}
				if mode == ModeOpen || mode == ModeSaveAs {
					a.exp.refresh(".", "")
					a.name.SetValue("capitulo-01.md")
				}

				lines := strings.Split(a.panel(), "\n")
				want := ansi.StringWidth(lines[0])

				for i, l := range lines {
					if got := ansi.StringWidth(l); got != want {
						t.Fatalf("%dx%d line %d is %d columns, want %d: %q",
							size.w, size.h, i, got, want, l)
					}
				}
				if want > size.w {
					t.Errorf("%dx%d panel is %d columns, wider than the terminal", size.w, size.h, want)
				}
				if len(lines) > size.h {
					t.Errorf("%dx%d panel is %d rows, taller than the terminal", size.w, size.h, len(lines))
				}
			})
		}
	}
}

// The whole screen is one string; a stray newline in it would push the status
// bar off the bottom of the terminal.
func TestTheScreenIsExactlyAsTallAsTheTerminal(t *testing.T) {
	for _, mode := range []Mode{ModeWrite, ModeHelp, ModeOpen} {
		a := testApp(t)
		a.mode = mode
		if mode == ModeOpen {
			a.exp.refresh(".", "")
		}

		if got := len(strings.Split(a.View(), "\n")); got != a.h {
			t.Errorf("mode %v: %d rows, want %d", mode, got, a.h)
		}
	}
}
