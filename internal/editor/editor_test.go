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

func backspaceN(e *Editor, n int) {
	for range n {
		e.Backspace()
	}
}

// A run of backspaces collapses into one undo step too, matching typing —
// deleting five characters used to take five separate undos to get the text
// back, not one.
func TestBackspaceTakesBackAWholeBurst(t *testing.T) {
	e := New()
	typeText(e, "hola mundo")

	backspaceN(e, 5) // removes "mundo"
	if got := e.Text(); got != "hola " {
		t.Fatalf("after 5 backspaces: %q, want %q", got, "hola ")
	}

	e.Undo()
	if got := e.Text(); got != "hola mundo" {
		t.Errorf("one undo after the backspaces: %q, want the whole burst back", got)
	}
}

// A backspace burst and a typing burst never merge into one undo step, even
// sitting right next to each other, or undo could reinsert text in the wrong
// place relative to what was typed after it.
func TestBackspaceThenTypingAreSeparateBursts(t *testing.T) {
	e := New()
	typeText(e, "abc")
	e.Backspace()    // "ab"
	typeText(e, "x") // "abx"

	e.Undo()
	if got := e.Text(); got != "ab" {
		t.Errorf("after undoing the typed x: %q, want %q", got, "ab")
	}

	e.Undo()
	if got := e.Text(); got != "abc" {
		t.Errorf("after undoing the backspace: %q, want %q", got, "abc")
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

// Passing through a short line used to permanently truncate the column: the
// next move down recomputed the wanted column from wherever the short line
// had just clamped the cursor to, instead of the column the user actually
// started from.
func TestVerticalMovementRemembersTheGoalColumn(t *testing.T) {
	e := New()
	e.SetText("una linea bastante larga\nla\nuna linea bastante larga tambien")
	vlines := e.VisualLines(100) // wide enough that each line is its own row

	e.cursor = 18 // column 18 on the first, long line

	e.MoveDown(vlines, false)
	if col := e.cursor - vlines[1].Start; col != 2 {
		t.Fatalf("on the short line: column %d, want 2 (clamped to its length)", col)
	}

	e.MoveDown(vlines, false)
	if col := e.cursor - vlines[2].Start; col != 18 {
		t.Errorf("past the short line: column %d, want 18 (the original goal)", col)
	}
}

// Any move that is not vertical drops the remembered goal, or the cursor
// would keep returning to a column the user has since moved away from.
func TestHorizontalMovementForgetsTheGoalColumn(t *testing.T) {
	e := New()
	e.SetText("una linea bastante larga\nla\nuna linea bastante larga tambien")
	vlines := e.VisualLines(100)

	e.cursor = 18
	e.MoveDown(vlines, false) // goal column is now 18, cursor clamped to column 2

	e.MoveLeft(false) // an explicit horizontal move

	e.MoveDown(vlines, false)
	if col := e.cursor - vlines[2].Start; col != 1 {
		t.Errorf("column %d, want 1 (from where MoveLeft actually left the cursor)", col)
	}
}

// DeleteWordLeft removes the same span WordLeft would move over, in one go.
func TestDeleteWordLeft(t *testing.T) {
	e := New()
	e.SetText("uno   dos tres")
	e.cursor = 14 // end of the buffer

	e.DeleteWordLeft()

	if got, want := e.Text(), "uno   dos "; got != want {
		t.Errorf("Text() = %q, want %q", got, want)
	}
	if e.cursor != 10 {
		t.Errorf("cursor = %d, want 10", e.cursor)
	}
}

// The deletion is one undo step, not one per character, the same guarantee
// Backspace gives a typing burst.
func TestDeleteWordLeftIsOneUndoStep(t *testing.T) {
	e := New()
	e.SetText("hola mundo")
	e.cursor = len(e.Runes())

	e.DeleteWordLeft()
	if got := e.Text(); got != "hola " {
		t.Fatalf("Text() = %q, want %q", got, "hola ")
	}

	e.Undo()
	if got := e.Text(); got != "hola mundo" {
		t.Errorf("Text() after undo = %q, want the word back in one step", got)
	}
}

// A live selection takes priority over the word boundary, matching
// Backspace and DeleteForward.
func TestDeleteWordLeftPrefersTheSelection(t *testing.T) {
	e := New()
	e.SetText("uno dos tres")
	e.cursor = 0
	e.MoveRight(true)
	e.MoveRight(true)
	e.MoveRight(true) // selects "uno"

	e.DeleteWordLeft()

	if got, want := e.Text(), " dos tres"; got != want {
		t.Errorf("Text() = %q, want %q", got, want)
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

// SessionWords reports what changed since the document was opened, not what
// the whole document holds.
func TestSessionWords(t *testing.T) {
	e := New()
	e.SetText("uno dos") // the moment the "document" is considered opened

	typeText(e, " tres")

	if got := e.SessionWords(); got != 1 {
		t.Errorf("SessionWords() = %d, want 1", got)
	}
}

// A session that mostly deletes reports a negative count rather than
// clamping at zero, so "wrote nothing" and "deleted a paragraph" don't look
// the same.
func TestSessionWordsCanGoNegative(t *testing.T) {
	e := New()
	e.SetText("uno dos tres") // cursor lands at the end

	e.DeleteWordLeft() // removes "tres"

	if got := e.SessionWords(); got != -1 {
		t.Errorf("SessionWords() = %d, want -1", got)
	}
}

func selection(t *testing.T, e *Editor) (start, end int) {
	t.Helper()
	start, end, ok := e.Selection()
	if !ok {
		t.Fatal("no selection, want a match")
	}
	return start, end
}

// Repeating a forward search steps to the next match instead of finding the
// one already selected, because the search starts at the cursor — which a
// match leaves at its own end.
func TestFindStepsToTheNextMatch(t *testing.T) {
	e := New()
	e.SetText("uno dos uno dos")
	e.cursor = 0

	if !e.Find("uno", false) {
		t.Fatal("Find did not match")
	}
	if start, end := selection(t, e); start != 0 || end != 3 {
		t.Errorf("selection (%d,%d), want (0,3)", start, end)
	}

	if !e.Find("uno", false) {
		t.Fatal("second Find did not match")
	}
	if start, end := selection(t, e); start != 8 || end != 11 {
		t.Errorf("selection (%d,%d), want (8,11) — the second uno", start, end)
	}
}

// A forward search that runs off the end wraps to the first match rather
// than reporting nothing left.
func TestFindWrapsForward(t *testing.T) {
	e := New()
	e.SetText("uno dos uno dos")
	e.cursor = len(e.Runes())

	if !e.Find("uno", false) {
		t.Fatal("Find did not match")
	}
	if start, end := selection(t, e); start != 0 || end != 3 {
		t.Errorf("selection (%d,%d), want (0,3) — wrapped to the first uno", start, end)
	}
}

// Backward search steps from the anchor, not the cursor, so it does not
// immediately re-select the match it is standing on.
func TestFindBackwardStepsToThePreviousMatch(t *testing.T) {
	e := New()
	e.SetText("uno dos uno dos")
	e.cursor = len(e.Runes())

	if !e.Find("uno", true) {
		t.Fatal("Find did not match")
	}
	if start, end := selection(t, e); start != 8 || end != 11 {
		t.Errorf("selection (%d,%d), want (8,11) — the last uno", start, end)
	}

	if !e.Find("uno", true) {
		t.Fatal("second backward Find did not match")
	}
	if start, end := selection(t, e); start != 0 || end != 3 {
		t.Errorf("selection (%d,%d), want (0,3) — wrapped to the first uno", start, end)
	}
}

func TestFindIsCaseInsensitive(t *testing.T) {
	e := New()
	e.SetText("Uno dos")

	if !e.Find("uno", false) {
		t.Fatal("Find did not match a different case")
	}
	if start, end := selection(t, e); start != 0 || end != 3 {
		t.Errorf("selection (%d,%d), want (0,3)", start, end)
	}
}

// Nothing to find leaves the cursor exactly where it was, rather than
// jumping to a leftover position from a stale search.
func TestFindWithNoMatchChangesNothing(t *testing.T) {
	e := New()
	e.SetText("uno dos")
	e.cursor = 2

	if e.Find("xyz", false) {
		t.Fatal("Find matched something that is not there")
	}
	if e.cursor != 2 {
		t.Errorf("cursor = %d, want unchanged 2", e.cursor)
	}
	if _, _, ok := e.Selection(); ok {
		t.Error("a failed search left a selection behind")
	}
}

// SetCursor is how ctrl+f puts the cursor back where it was when the search
// is cancelled, rather than wherever the last match left it.
func TestSetCursorClampsAndClearsSelection(t *testing.T) {
	e := New()
	e.SetText("uno dos")
	e.Find("dos", false) // leaves a selection

	e.SetCursor(2)

	if e.Cursor() != 2 {
		t.Errorf("Cursor() = %d, want 2", e.Cursor())
	}
	if _, _, ok := e.Selection(); ok {
		t.Error("SetCursor left a selection behind")
	}

	e.SetCursor(999)
	if want := len(e.Runes()); e.Cursor() != want {
		t.Errorf("Cursor() = %d, want %d (clamped)", e.Cursor(), want)
	}
}
