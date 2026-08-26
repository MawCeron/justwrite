package ui

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestPreview writes one frame per screen to assets/frames/*.ans, which
// assets/screens.sh turns into the SVGs in the README. It is skipped unless
// PREVIEW=1, so an ordinary `go test ./...` never writes anything.
//
// These frames are the real View() output, byte for byte what the terminal is
// handed — not a mock-up. What they are not is a recording of a live PTY; the
// animated demo still needs VHS.
func TestPreview(t *testing.T) {
	if os.Getenv("PREVIEW") == "" {
		t.Skip("set PREVIEW=1 to write assets/frames")
	}
	// go test hands the binary a pipe, so lipgloss would drop every colour.
	lipgloss.SetColorProfile(termenv.TrueColor)

	// Anchored to this source file, not the working directory: a compiled test
	// binary run from the repo root would otherwise write two levels above it.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the source file to write beside")
	}
	out := filepath.Join(filepath.Dir(thisFile), "..", "..", "assets", "frames")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	write := func(name string, a App) {
		t.Helper()
		path := filepath.Join(out, name+".ans")
		if err := os.WriteFile(path, []byte(a.View()+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", path)
	}

	// Wide enough that the page is visibly narrower than the terminal: the
	// measure caps at 82 columns however much room there is, and a screenshot
	// at exactly 82 would hide that.
	const w, h = 104, 30

	scene := func() App {
		a := testApp(t) // a real App, built the way main builds it
		a.w, a.h = w, h
		a.ed.SetText(lorem)
		a.ed.Path = "lorem.md"
		a.ed.Modified = true
		return a
	}

	write("writing", scene())

	help := scene()
	help.mode = ModeHelp
	write("help", help)

	about := scene()
	about.mode = ModeAbout
	write("about", about)

	stats := scene()
	stats.mode = ModeStats
	write("stats", stats)

	confirm := scene()
	confirm.mode = ModeConfirm
	confirm.confirm = confirmation{message: "discard changes?"}
	write("confirm", confirm)

	conflict := scene()
	conflict.mode = ModeConflict
	write("conflict", conflict)

	find := scene()
	find.mode = ModeFind
	find.onFindQuery = true
	find.findInput.SetValue("cursor")
	find.findInput.CursorEnd()
	find.findInput.Focus()
	write("find", find)

	dialog := scene()
	dialog.mode = ModeOpen
	dialog.exp.refresh(fixtureDir(t), "")
	// The listing is real, read off disk. Only the label is staged: the true
	// path is a throwaway temp directory, and putting it on camera would both
	// look like noise and publish somebody's home directory layout.
	dialog.exp.dir = "/home/you/writing"
	write("dialog", dialog)
}

// fixtureDir builds a directory that looks like somewhere a book is being
// written, including the two kinds of file the listing filters out.
func fixtureDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	if err := os.Mkdir(filepath.Join(dir, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{
		"chapter-01.md", "chapter-02.md", "chapter-03.md",
		"outline.md", "characters.md", "research.txt",
	} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Neither of these should appear in the shot: that is the point of them.
	if err := os.WriteFile(filepath.Join(dir, ".draft.swp"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cover.png"), []byte("\x89PNG\x00\x1a"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const lorem = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod " +
	"tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis " +
	"nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.\n\n" +
	"Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu " +
	"fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in " +
	"culpa qui officia deserunt mollit anim id est laborum.\n\n" +
	"Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium " +
	"doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore " +
	"veritatis et quasi architecto beatae vitae dicta sunt explicabo."
