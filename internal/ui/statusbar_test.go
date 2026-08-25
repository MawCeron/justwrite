package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// The bar is painted end to end, so it has to measure exactly the width of the
// terminal. One column over and it wraps into a second line, shoving the page
// up; one under and the paint stops short of the edge.
func TestTheBarIsExactlyAsWideAsTheTerminal(t *testing.T) {
	for _, c := range []struct {
		name string
		path string
		w    int
	}{
		{"no document yet", "", 80},
		{"a short name", "/home/m/nota.md", 80},
		{"a name that has to be cut", "/home/m/" + strings.Repeat("nombre-larguisimo", 12) + ".md", 80},
		{"a narrow terminal", "/home/m/nota.md", 30},
		{"a wide terminal", "/home/m/nota.md", 200},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := testApp(t)
			a.w = c.w
			a.ed.Path = c.path

			bar := a.statusBar()

			if got := ansi.StringWidth(bar); got != c.w {
				t.Errorf("bar is %d columns wide, want %d", got, c.w)
			}
			if strings.Contains(bar, "\n") {
				t.Error("the bar broke into more than one line")
			}
		})
	}
}

// An unsaved document has no name to show, and inventing one ("untitled") is
// noise: the marker alone says everything there is to say.
func TestWhatTheBarSaysAboutTheDocument(t *testing.T) {
	for _, c := range []struct {
		name     string
		path     string
		modified bool
		want     string
	}{
		{"a new document says nothing", "", false, ""},
		{"a new document with changes shows only the marker", "", true, "*"},
		{"a saved file shows its name", "/home/m/nota.md", false, "nota.md"},
		{"an edited file marks it", "/home/m/nota.md", true, "nota.md*"},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := testApp(t)
			a.ed.Path = c.path
			a.ed.Modified = c.modified

			if got := a.note(); got != c.want {
				t.Errorf("note = %q, want %q", got, c.want)
			}
		})
	}
}

// The chip has room for a tag, not for the pseudo-version Go makes up from the
// commit when there is no tag.
func TestReleaseVersion(t *testing.T) {
	for in, want := range map[string]string{
		"":                                     "dev",
		"(devel)":                              "dev",
		"v0.0.0-20260810012633-966b7937475d":   "dev",
		"v0.1.1-0.20260810012633-966b7937475d": "dev",
		"v0.1.0":                               "v0.1.0",
		"v1.2.3":                               "v1.2.3",
	} {
		if got := releaseVersion(in); got != want {
			t.Errorf("releaseVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

// A status message no longer hides which file it is about — only when there
// is no name yet does the message stand alone.
func TestAMessageJoinsTheFilename(t *testing.T) {
	a := testApp(t)
	a.ed.Path = "/home/m/nota.md"
	a.status = "saved"

	if got, want := a.note(), "nota.md · saved"; got != want {
		t.Errorf("note = %q, want %q", got, want)
	}

	a.ed.Path = ""
	if got, want := a.note(), "saved"; got != want {
		t.Errorf("note = %q, want %q (no filename yet)", got, want)
	}
}
