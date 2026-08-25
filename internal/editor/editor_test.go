package editor

import "testing"

func typeText(e *Editor, s string) {
	for _, r := range s {
		e.InsertRune(r)
	}
}

// Typing is collapsed into bursts so that ctrl+z walks back word by word. A
// per-letter undo makes the history useless for writing prose.
func TestUndoTakesBackAWholeBurst(t *testing.T) {
	e := New()
	typeText(e, "abc")

	e.Undo()

	if got := e.Text(); got != "" {
		t.Errorf("one undo left %q, want the whole burst gone", got)
	}
}

func TestSpaceEndsTheBurst(t *testing.T) {
	e := New()
	typeText(e, "hola mundo")

	e.Undo()
	if got := e.Text(); got != "hola" {
		t.Errorf("after one undo: %q, want %q", got, "hola")
	}

	e.Undo()
	if got := e.Text(); got != "" {
		t.Errorf("after two undos: %q, want empty", got)
	}
}

// A newline is its own step, so undo never merges two paragraphs back together.
func TestNewlineIsItsOwnUndoStep(t *testing.T) {
	e := New()
	typeText(e, "uno\ndos")

	e.Undo()
	if got := e.Text(); got != "uno\n" {
		t.Errorf("after one undo: %q, want %q", got, "uno\n")
	}

	e.Undo()
	if got := e.Text(); got != "uno" {
		t.Errorf("after two undos: %q, want %q", got, "uno")
	}
}

// Typing over a selection is two changes — the delete and the insert — so it
// takes two undos to get the replaced text back. Losing the deleted text would
// be the real bug.
func TestTypingOverASelectionCanBeUndone(t *testing.T) {
	e := New()
	typeText(e, "hello")
	e.SelectAll()

	e.InsertRune('X')
	if got := e.Text(); got != "X" {
		t.Fatalf("overtype produced %q, want %q", got, "X")
	}

	e.Undo()
	e.Undo()
	if got := e.Text(); got != "hello" {
		t.Errorf("the replaced text did not come back: %q", got)
	}
}

func TestRedoReappliesAnUndo(t *testing.T) {
	e := New()
	typeText(e, "abc")
	e.Undo()

	e.Redo()

	if got := e.Text(); got != "abc" {
		t.Errorf("redo produced %q, want %q", got, "abc")
	}
}

func TestCutThenPaste(t *testing.T) {
	e := New()
	typeText(e, "hola mundo")
	e.cursor, e.anchor = 10, 4 // " mundo"

	if got := e.Cut(); got != " mundo" {
		t.Fatalf("cut returned %q", got)
	}
	if got := e.Text(); got != "hola" {
		t.Fatalf("cut left %q", got)
	}

	e.Paste("")

	if got := e.Text(); got != "hola mundo" {
		t.Errorf("paste produced %q", got)
	}
	if e.cursor != 10 {
		t.Errorf("cursor at %d after paste, want 10 (end of pasted text)", e.cursor)
	}
}

// External text wins over the internal clipboard: it is what the terminal just
// handed us in a bracketed paste, and it is what the user actually meant.
func TestPastePrefersExternalText(t *testing.T) {
	e := New()
	typeText(e, "ab")
	e.SelectAll()
	e.Copy()
	e.ClearSelection()

	e.Paste("external")

	if got := e.Text(); got != "abexternal" {
		t.Errorf("got %q, want %q", got, "abexternal")
	}
}

func TestWordMovement(t *testing.T) {
	e := New()
	e.SetText("uno   dos tres")

	for _, c := range []struct {
		name  string
		start int
		move  func()
		want  int
	}{
		{"left from the end lands at the last word", 14, func() { e.WordLeft(false) }, 10},
		{"left skips a run of spaces", 6, func() { e.WordLeft(false) }, 0},
		{"left at the start stays put", 0, func() { e.WordLeft(false) }, 0},
		{"right from the start ends the first word", 0, func() { e.WordRight(false) }, 3},
		{"right skips a run of spaces", 3, func() { e.WordRight(false) }, 9},
		{"right at the end stays put", 14, func() { e.WordRight(false) }, 14},
	} {
		e.cursor = c.start
		c.move()
		if e.cursor != c.want {
			t.Errorf("%s: cursor %d, want %d", c.name, e.cursor, c.want)
		}
	}
}

// PageUp used to nudge Scroll by hand while PageDown left it entirely to
// AdjustScroll, so paging down and back up landed the viewport one extra
// scrollMargin higher than paging down and then walking back up would.
func TestPagingUpAndDownIsSymmetric(t *testing.T) {
	const page = 10
	e := fiftyLines()
	vlines := e.VisualLines(20)
	e.cursor = vlines[20].Start
	e.AdjustScroll(vlines, page)

	e.PageDown(vlines, page, false)
	e.AdjustScroll(vlines, page)

	e.PageUp(vlines, page, false)
	e.AdjustScroll(vlines, page)

	// Back near line 20, under the same "keep a margin above the cursor"
	// rule AdjustScroll already applies to any upward move: 20-scrollMargin.
	if want := 20 - scrollMargin; e.Scroll != want {
		t.Errorf("Scroll = %d after paging down then up, want %d", e.Scroll, want)
	}
}

// Home and End only reach the edges of the current visual line; ctrl+home and
// ctrl+end have to reach the edges of the whole document instead.
func TestDocumentHomeAndEnd(t *testing.T) {
	e := New()
	e.SetText("uno\ndos\ntres")
	e.cursor = 6 // somewhere in the middle line

	e.DocumentEnd(false)
	if e.cursor != len(e.Runes()) {
		t.Errorf("DocumentEnd: cursor = %d, want %d", e.cursor, len(e.Runes()))
	}

	e.DocumentHome(false)
	if e.cursor != 0 {
		t.Errorf("DocumentHome: cursor = %d, want 0", e.cursor)
	}
}

// ctrl+shift+home/end select like every other movement key: the anchor stays
// where the selection began.
func TestDocumentHomeAndEndExtendSelection(t *testing.T) {
	e := New()
	e.SetText("uno\ndos\ntres")
	e.cursor = 6

	e.DocumentEnd(true)

	start, end, ok := e.Selection()
	if !ok || start != 6 || end != len(e.Runes()) {
		t.Errorf("selection (%d,%d,%v), want (6,%d,true)", start, end, ok, len(e.Runes()))
	}
}

// Shift+arrow has to keep the anchor where the selection began, or extending a
// selection would only ever select one character.
func TestExtendingKeepsTheAnchor(t *testing.T) {
	e := New()
	e.SetText("abcdef")
	e.cursor = 2

	e.MoveRight(true)
	e.MoveRight(true)

	start, end, ok := e.Selection()
	if !ok || start != 2 || end != 4 {
		t.Errorf("selection (%d,%d,%v), want (2,4,true)", start, end, ok)
	}
}

func TestStats(t *testing.T) {
	e := New()
	e.SetText("uno dos\ntres")

	if got := e.WordCount(); got != 3 {
		t.Errorf("WordCount = %d, want 3", got)
	}
	if got := e.CharCount(); got != 11 { // the newline does not count
		t.Errorf("CharCount = %d, want 11", got)
	}
	// "under a minute" is reserved for an empty document; anything with words
	// in it rounds up to at least one minute.
	if mins, underOne := e.ReadingTime(); mins != 1 || underOne {
		t.Errorf("ReadingTime = (%d, %v), want (1, false)", mins, underOne)
	}
	if mins, underOne := New().ReadingTime(); mins != 0 || !underOne {
		t.Errorf("empty ReadingTime = (%d, %v), want (0, true)", mins, underOne)
	}
}
