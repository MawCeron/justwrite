package ui

import (
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/charmbracelet/x/ansi"
)

const appName = "justwrite"

// BuildInfo reads the version and commit Go stamps into the binary. A plain
// `go build` inside the repository records the revision, and `go install
// module@version` records the tag, so there is nothing to keep up to date by
// hand and no -ldflags to remember.
var BuildInfo = sync.OnceValues(func() (version, commit string) {
	version, commit = "dev", ""
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version, commit
	}
	version = releaseVersion(info.Main.Version)

	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			commit = s.Value
			if len(commit) > 7 {
				commit = commit[:7]
			}
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if dirty && commit != "" {
		commit += "+" // built from a working tree with uncommitted changes
	}
	return version, commit
})

// releaseVersion is what the chip is willing to call itself. Anything that is
// not a plain tag — no version at all, "(devel)", or the pseudo-version Go
// derives from the commit when there is no tag (v0.0.0-20260810012633-966b79…)
// — is just "dev": the commit hash beside it in About is the real identity, and
// forty columns of pseudo-version in an eight-column chip says nothing.
func releaseVersion(v string) string {
	if v == "" || v == "(devel)" || strings.Contains(v, "-") {
		return "dev"
	}
	return v
}

func versionLabel() string {
	v, _ := BuildInfo()
	return v
}

// VersionString is what justwrite calls itself: the tag it was built from,
// or "dev", plus the commit when one is known. Shared by --version and the
// About panel, so the two can never drift apart.
func VersionString() string {
	v, commit := BuildInfo()
	s := appName + " " + v
	if commit != "" {
		s += " (" + commit + ")"
	}
	return s
}

// statusBar draws the one line of chrome justwrite has, at the full width of
// the terminal: version chip, document, painted gap, help chip.
func (a App) statusBar() string {
	noteSt, chipSt := noteStyle, helpChipStyle
	switch {
	case a.statusErr:
		noteSt, chipSt = errNoteStyle, errChipStyle
	case a.status != "":
		noteSt, chipSt = okNoteStyle, okChipStyle
	}

	chip := versionStyle.Render(" " + versionLabel() + " ")
	help := chipSt.Render(" F1 Help ")

	room := a.w - ansi.StringWidth(chip) - ansi.StringWidth(help)
	note := a.note()
	if note != "" {
		note = " " + note + " "
	}
	note = ansi.Truncate(note, max(room, 0), "…")

	gap := strings.Repeat(" ", max(room-ansi.StringWidth(note), 0))
	return chip + noteSt.Render(note+gap) + help
}

// note is what the bar says about the document. A document that has never been
// saved shows no invented name — only the marker, and only once it has changes
// worth losing. A status message joins the name rather than hiding it, so the
// bar never stops saying which file you are in.
func (a App) note() string {
	name := ""
	if a.ed.Path != "" {
		name = filepath.Base(a.ed.Path)
	}
	if a.ed.Modified {
		name += "*"
	}
	switch {
	case a.status == "":
		return name
	case name == "":
		return a.status
	default:
		return name + " · " + a.status
	}
}
