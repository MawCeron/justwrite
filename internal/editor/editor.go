// Package editor holds the document: the text, the cursor, the selection and
// the undo history. It knows nothing about terminals or Bubble Tea, so every
// rule in here can be tested by calling a method and reading the result.
package editor

import (
	"slices"
	"strings"
	"time"
	"unicode"
)

// noSel marks "no selection" in anchor. The document uses rune indexes
// throughout — never byte indexes — so a cursor never lands mid-character.
const noSel = -1

// noGoal marks "no remembered column" in goalCol.
const noGoal = -1

// editOp is one undoable change. insert=true means text was put in at at;
// insert=false means text was taken out from at. Undo applies the opposite.
// seq is when it was created — never reused, even when undo and redo move
// the same op back and forth — so Modified can tell a genuine return to the
// saved state apart from a diverged history that only happens to be the same
// number of steps deep.
type editOp struct {
	insert bool
	at     int
	text   string
	seq    int
}

type Editor struct {
	buf    []rune
	cursor int
	anchor int
	crlf   bool // true if the loaded file used \r\n; Save() restores it

	// goalCol is the column consecutive vertical moves try to return to, so
	// passing through a short line does not permanently forget where a
	// longer line above or below it was. noGoal until the first vertical
	// move, and invalidated by anything that is not one.
	goalCol int

	Path     string // "" until the document has been saved somewhere
	Modified bool
	Scroll   int

	// loadedModTime is what Save compares the file's mtime against to catch
	// a change from outside justwrite. The zero value means the file did not
	// exist the last time this was recorded.
	loadedModTime time.Time

	clipboard string

	undo  []editOp
	redo  []editOp
	seq   int // next sequence number to hand out to a new edit
	saved int // the seq on top of undo when the document was last loaded or saved

	// A burst of typed characters, or a burst of contiguous backspaces,
	// collapses into a single undo step, so one ctrl+z takes back a word
	// rather than a letter. pendingDelete tells the two kinds of burst apart
	// — they never merge into one op even when adjacent. Committed by
	// anything that breaks it: a space, a newline, a move, a save.
	pendingAt     int
	pendingText   []rune
	pendingDelete bool

	// openWords is the word count at the moment the document was last loaded
	// or reset, so the stats panel can report what changed this session
	// rather than what the whole document holds.
	openWords int
}

func New() *Editor {
	return &Editor{anchor: noSel, pendingAt: noSel, goalCol: noGoal}
}

func (e *Editor) Text() string  { return string(e.buf) }
func (e *Editor) Runes() []rune { return e.buf }
func (e *Editor) Cursor() int   { return e.cursor }
func (e *Editor) Len() int      { return len(e.buf) }

// SetText replaces the document wholesale, as when a file is opened. The
// history belongs to the old document and is dropped with it.
func (e *Editor) SetText(s string) {
	e.buf = []rune(s)
	e.cursor = len(e.buf)
	e.anchor = noSel
	e.Scroll = 0
	e.crlf = false
	e.goalCol = noGoal
	e.undo = nil
	e.redo = nil
	e.seq = 0
	e.saved = 0
	e.pendingAt = noSel
	e.pendingText = nil
	e.pendingDelete = false
	e.openWords = e.WordCount()
	e.loadedModTime = time.Time{}
}

// NewDocument resets to a blank, unnamed document.
func (e *Editor) NewDocument() {
	e.SetText("")
	e.cursor = 0
	e.Path = ""
	e.Modified = false
}

// ─── Selection ───────────────────────────────────────────────────────────────

// Selection returns the ordered range and whether there is one at all.
func (e *Editor) Selection() (start, end int, ok bool) {
	if e.anchor == noSel {
		return 0, 0, false
	}
	if e.anchor <= e.cursor {
		return e.anchor, e.cursor, true
	}
	return e.cursor, e.anchor, true
}

func (e *Editor) ClearSelection() { e.anchor = noSel }

func (e *Editor) startSelection() {
	if e.anchor == noSel {
		e.anchor = e.cursor
	}
}

func (e *Editor) SelectAll() {
	e.CommitPending()
	e.anchor = 0
	e.cursor = len(e.buf)
}

// deleteSelection removes the selected text and returns it, leaving the cursor
// where the selection started.
func (e *Editor) deleteSelection() string {
	start, end, ok := e.Selection()
	if !ok {
		return ""
	}
	text := string(e.buf[start:end])
	e.buf = slices.Delete(e.buf, start, end)
	e.cursor = start
	e.anchor = noSel
	e.Modified = true
	return text
}

// replaceSelection deletes the selection as its own undo step, so undoing an
// overtype brings the replaced text back.
func (e *Editor) replaceSelection() {
	if _, _, ok := e.Selection(); !ok {
		return
	}
	e.CommitPending()
	if deleted := e.deleteSelection(); deleted != "" {
		e.push(editOp{at: e.cursor, text: deleted})
	}
}

// ─── Clipboard ───────────────────────────────────────────────────────────────

// Copy returns the selected text, also keeping it for a later Paste. The
// caller is responsible for handing it to the system clipboard.
func (e *Editor) Copy() string {
	start, end, ok := e.Selection()
	if !ok {
		return ""
	}
	e.clipboard = string(e.buf[start:end])
	return e.clipboard
}

func (e *Editor) Cut() string {
	e.CommitPending()
	start, end, ok := e.Selection()
	if !ok {
		return ""
	}
	text := string(e.buf[start:end])
	e.clipboard = text
	e.push(editOp{at: start, text: text})
	e.buf = slices.Delete(e.buf, start, end)
	e.cursor = start
	e.anchor = noSel
	e.Modified = true
	return text
}

// Paste inserts external text when there is any — a bracketed paste from the
// terminal — and otherwise falls back to what was last copied inside justwrite.
func (e *Editor) Paste(external string) {
	e.CommitPending()
	text := external
	if text == "" {
		text = e.clipboard
	}
	if text == "" {
		return
	}
	e.replaceSelection()
	e.insertAsOneOp(text)
}

// ─── Undo / redo ─────────────────────────────────────────────────────────────

// push adds a newly-made edit to the undo stack, tagged with a fresh
// sequence number, and retires whatever redo history came before it — a
// fresh edit is a new branch, not a continuation of one that was undone.
func (e *Editor) push(op editOp) {
	e.seq++
	op.seq = e.seq
	e.undo = append(e.undo, op)
	e.redo = nil
}

// currentSeq identifies the document's current state: the seq on top of the
// undo stack, or 0 — the same number a document starts at — once there is
// nothing left to undo.
func (e *Editor) currentSeq() int {
	if len(e.undo) == 0 {
		return 0
	}
	return e.undo[len(e.undo)-1].seq
}

// CommitPending flushes the current burst — typed or backspaced — into the
// undo stack. It must be called before anything that breaks the burst, or
// the burst would keep growing across unrelated edits.
func (e *Editor) CommitPending() {
	if e.pendingAt == noSel {
		return
	}
	if len(e.pendingText) > 0 {
		e.push(editOp{insert: !e.pendingDelete, at: e.pendingAt, text: string(e.pendingText)})
	}
	e.pendingAt = noSel
	e.pendingText = nil
	e.pendingDelete = false
}

func (e *Editor) Undo() {
	e.CommitPending()
	if len(e.undo) == 0 {
		return
	}
	op := e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]
	e.apply(op, true)
	e.redo = append(e.redo, op)
}

func (e *Editor) Redo() {
	e.CommitPending()
	if len(e.redo) == 0 {
		return
	}
	op := e.redo[len(e.redo)-1]
	e.redo = e.redo[:len(e.redo)-1]
	e.undo = append(e.undo, op)
	e.apply(op, false)
}

// apply runs op, or its opposite when reverse is set. Undo and redo only ever
// move an existing op between the two stacks, so e.undo already reflects the
// resulting position by the time this runs — comparing it against saved is
// what tells a genuine return to the saved state apart from an edit that
// merely landed on the same undo depth.
func (e *Editor) apply(op editOp, reverse bool) {
	rs := []rune(op.text)
	if op.insert == reverse {
		// Take the text back out.
		end := min(op.at+len(rs), len(e.buf))
		e.buf = slices.Delete(e.buf, min(op.at, end), end)
		e.cursor = op.at
	} else {
		e.buf = slices.Insert(e.buf, min(op.at, len(e.buf)), rs...)
		e.cursor = op.at + len(rs)
	}
	e.anchor = noSel
	e.Modified = e.currentSeq() != e.saved
}

// ─── Editing ─────────────────────────────────────────────────────────────────

func (e *Editor) InsertRune(r rune) {
	e.replaceSelection()

	// A newline is always its own undo step, and a space closes the word
	// before it, so undo walks back word by word instead of letter by letter.
	if r == '\n' {
		e.CommitPending()
		e.insertAsOneOp("\n")
		return
	}
	if r == ' ' {
		e.CommitPending()
	}

	at := e.cursor
	e.buf = slices.Insert(e.buf, at, r)
	e.cursor++
	e.Modified = true
	e.redo = nil

	if !e.pendingDelete && e.pendingAt != noSel && e.pendingAt+len(e.pendingText) == at {
		e.pendingText = append(e.pendingText, r)
		return
	}
	e.CommitPending()
	e.pendingAt, e.pendingText, e.pendingDelete = at, []rune{r}, false
}

// InsertString inserts text as a single undo step — a tab, or a paste.
func (e *Editor) InsertString(s string) {
	e.CommitPending()
	e.replaceSelection()
	e.insertAsOneOp(s)
}

func (e *Editor) insertAsOneOp(s string) {
	rs := []rune(s)
	if len(rs) == 0 {
		return
	}
	at := e.cursor
	e.buf = slices.Insert(e.buf, at, rs...)
	e.cursor += len(rs)
	e.Modified = true
	e.push(editOp{insert: true, at: at, text: s})
}

// Backspace collapses a run of contiguous backspaces into one undo step, the
// same as typing does — deleting five characters after typing "hola mundo"
// used to take seven undos to reach an empty document instead of two.
func (e *Editor) Backspace() {
	if _, _, ok := e.Selection(); ok {
		e.replaceSelection()
		return
	}
	if e.cursor == 0 {
		return
	}
	r := e.buf[e.cursor-1]
	e.buf = slices.Delete(e.buf, e.cursor-1, e.cursor)
	e.cursor--
	e.Modified = true
	e.redo = nil

	if e.pendingDelete && e.pendingAt == e.cursor+1 {
		e.pendingText = append([]rune{r}, e.pendingText...)
		e.pendingAt = e.cursor
		return
	}
	e.CommitPending()
	e.pendingAt, e.pendingText, e.pendingDelete = e.cursor, []rune{r}, true
}

func (e *Editor) DeleteForward() {
	e.CommitPending()
	if _, _, ok := e.Selection(); ok {
		e.replaceSelection()
		return
	}
	if e.cursor >= len(e.buf) {
		return
	}
	r := e.buf[e.cursor]
	e.buf = slices.Delete(e.buf, e.cursor, e.cursor+1)
	e.Modified = true
	e.push(editOp{at: e.cursor, text: string(r)})
}

// ─── Navigation ──────────────────────────────────────────────────────────────

// beforeMove ends the typing burst and either extends or drops the
// selection. A non-vertical move also drops the remembered goal column, so
// the next up/down/page starts fresh from wherever the cursor actually is
// instead of somewhere a move before it was heading.
func (e *Editor) beforeMove(extend, vertical bool) {
	e.CommitPending()
	if !vertical {
		e.goalCol = noGoal
	}
	if extend {
		e.startSelection()
	} else {
		e.ClearSelection()
	}
}

// verticalCol is the column a vertical move should land on: the remembered
// goal from an earlier vertical move in the same run, or the cursor's
// current column when this is the first of the run — which becomes the goal
// for whatever comes after it.
//
// ponytail: only moves routed through beforeMove invalidate the goal; typing
// or deleting through a short line does not. Narrow sequence to hit (move
// vertically, edit, move vertically again with nothing else in between) —
// clear goalCol at the top of the editing entry points too if that turns out
// to matter in practice.
func (e *Editor) verticalCol(cur int, vlines []VisualLine) int {
	if e.goalCol == noGoal {
		e.goalCol = e.cursor - vlines[cur].Start
	}
	return e.goalCol
}

func (e *Editor) MoveLeft(extend bool) {
	e.beforeMove(extend, false)
	if e.cursor > 0 {
		e.cursor--
	}
}

func (e *Editor) MoveRight(extend bool) {
	e.beforeMove(extend, false)
	if e.cursor < len(e.buf) {
		e.cursor++
	}
}

func (e *Editor) MoveUp(vlines []VisualLine, extend bool) {
	e.beforeMove(extend, true)
	cur := e.CursorVisualLine(vlines)
	if cur == 0 || len(vlines) == 0 {
		return
	}
	e.cursor = columnIn(vlines[cur-1], e.verticalCol(cur, vlines))
}

func (e *Editor) MoveDown(vlines []VisualLine, extend bool) {
	e.beforeMove(extend, true)
	cur := e.CursorVisualLine(vlines)
	if cur+1 >= len(vlines) {
		return
	}
	e.cursor = columnIn(vlines[cur+1], e.verticalCol(cur, vlines))
}

func (e *Editor) Home(vlines []VisualLine, extend bool) {
	e.beforeMove(extend, false)
	if len(vlines) == 0 {
		return
	}
	e.cursor = vlines[e.CursorVisualLine(vlines)].Start
}

func (e *Editor) End(vlines []VisualLine, extend bool) {
	e.beforeMove(extend, false)
	if len(vlines) == 0 {
		return
	}
	e.cursor = vlines[e.CursorVisualLine(vlines)].End
}

// DocumentHome moves to the very start of the document — ctrl+home. Home only
// reaches the start of the current visual line.
func (e *Editor) DocumentHome(extend bool) {
	e.beforeMove(extend, false)
	e.cursor = 0
}

// DocumentEnd moves to the very end of the document — ctrl+end.
func (e *Editor) DocumentEnd(extend bool) {
	e.beforeMove(extend, false)
	e.cursor = len(e.buf)
}

func (e *Editor) PageUp(vlines []VisualLine, page int, extend bool) {
	e.beforeMove(extend, true)
	if len(vlines) == 0 {
		return
	}
	cur := e.CursorVisualLine(vlines)
	col := e.verticalCol(cur, vlines)
	e.cursor = columnIn(vlines[max(cur-page, 0)], col)
}

func (e *Editor) PageDown(vlines []VisualLine, page int, extend bool) {
	e.beforeMove(extend, true)
	if len(vlines) == 0 {
		return
	}
	cur := e.CursorVisualLine(vlines)
	col := e.verticalCol(cur, vlines)
	e.cursor = columnIn(vlines[min(cur+page, len(vlines)-1)], col)
}

// columnIn keeps the wanted column when it fits, and clamps to the end of a
// shorter line otherwise.
func columnIn(vl VisualLine, col int) int {
	return vl.Start + min(col, vl.End-vl.Start)
}

// wordBoundaryLeft is where a word starts, scanning left from pos: past any
// whitespace immediately before it, then past the word itself.
func (e *Editor) wordBoundaryLeft(pos int) int {
	for pos > 0 && unicode.IsSpace(e.buf[pos-1]) {
		pos--
	}
	for pos > 0 && !unicode.IsSpace(e.buf[pos-1]) {
		pos--
	}
	return pos
}

func (e *Editor) WordLeft(extend bool) {
	e.beforeMove(extend, false)
	e.cursor = e.wordBoundaryLeft(e.cursor)
}

func (e *Editor) WordRight(extend bool) {
	e.beforeMove(extend, false)
	pos := e.cursor
	for pos < len(e.buf) && unicode.IsSpace(e.buf[pos]) {
		pos++
	}
	for pos < len(e.buf) && !unicode.IsSpace(e.buf[pos]) {
		pos++
	}
	e.cursor = pos
}

// DeleteWordLeft removes the word before the cursor as a single undo step —
// ctrl+w, the readline binding for the same thing. ctrl+backspace was meant
// to be the other way in, but without the Kitty keyboard protocol a terminal
// sends the identical byte for it and for a plain backspace, so there is
// nothing to bind it to. A live selection is deleted instead of consulting
// the word boundary, matching Backspace and DeleteForward.
func (e *Editor) DeleteWordLeft() {
	e.CommitPending()
	if _, _, ok := e.Selection(); ok {
		e.replaceSelection()
		return
	}
	start := e.wordBoundaryLeft(e.cursor)
	if start == e.cursor {
		return
	}
	text := string(e.buf[start:e.cursor])
	e.buf = slices.Delete(e.buf, start, e.cursor)
	e.cursor = start
	e.Modified = true
	e.push(editOp{at: start, text: text})
}

// ─── Find ────────────────────────────────────────────────────────────────────

// Find selects the next case-insensitive occurrence of query — the previous
// one, when backward is set — wrapping around the document when the search
// runs off one end. Reports whether anything matched; the cursor and
// selection are left alone when nothing does.
//
// A match already selected from an earlier Find is excluded from the same
// search: forward looks from the cursor, which a match leaves at its own
// end, and backward looks from the anchor, which sits at its own start —
// so repeating the search steps to the next one instead of finding itself.
func (e *Editor) Find(query string, backward bool) bool {
	needle := []rune(strings.ToLower(query))
	if len(needle) == 0 {
		return false
	}
	e.CommitPending()
	e.goalCol = noGoal

	hay := []rune(strings.ToLower(string(e.buf)))
	from := e.cursor
	if backward {
		if _, _, ok := e.Selection(); ok {
			from = e.anchor
		}
	}

	pos, found := 0, false
	if backward {
		if pos, found = lastRuneIndex(hay, needle, from); !found {
			pos, found = lastRuneIndex(hay, needle, len(hay))
		}
	} else {
		if pos, found = firstRuneIndex(hay, needle, from); !found {
			pos, found = firstRuneIndex(hay, needle, 0)
		}
	}
	if !found {
		return false
	}
	e.anchor = pos
	e.cursor = pos + len(needle)
	return true
}

// SetCursor moves the cursor to pos directly, clamped to the buffer and
// clearing any selection — putting the cursor back where ctrl+f found it
// when the search is cancelled, not wherever the last match left it.
func (e *Editor) SetCursor(pos int) {
	e.cursor = max(0, min(pos, len(e.buf)))
	e.anchor = noSel
	e.goalCol = noGoal
}

// firstRuneIndex is the index of the first occurrence of needle in hay at or
// after from, or false. A document is small enough that this being a linear
// scan rather than a real string-search algorithm is not worth the code.
func firstRuneIndex(hay, needle []rune, from int) (int, bool) {
	for i := max(from, 0); i+len(needle) <= len(hay); i++ {
		if slices.Equal(hay[i:i+len(needle)], needle) {
			return i, true
		}
	}
	return 0, false
}

// lastRuneIndex is the index of the last occurrence of needle in hay ending
// at or before from, or false.
func lastRuneIndex(hay, needle []rune, from int) (int, bool) {
	for i := min(from, len(hay)) - len(needle); i >= 0; i-- {
		if slices.Equal(hay[i:i+len(needle)], needle) {
			return i, true
		}
	}
	return 0, false
}

// ─── Stats ───────────────────────────────────────────────────────────────────

func (e *Editor) WordCount() int { return len(strings.Fields(string(e.buf))) }

// SessionWords is how many words have been added since the document was
// opened — negative if the session was mostly deleting rather than writing.
func (e *Editor) SessionWords() int { return e.WordCount() - e.openWords }

func (e *Editor) CharCount() int {
	n := 0
	for _, r := range e.buf {
		if r != '\n' {
			n++
		}
	}
	return n
}

func (e *Editor) PageCount(textHeight, textWidth int) int {
	if textHeight <= 0 || textWidth <= 0 {
		return 1
	}
	total := max(len(e.VisualLines(textWidth)), 1)
	return (total + textHeight - 1) / textHeight
}

// ReadingTime reports minutes at 200 wpm, and whether it rounds to under one.
func (e *Editor) ReadingTime() (minutes int, underOne bool) {
	words := e.WordCount()
	if words == 0 {
		return 0, true
	}
	return (words + 199) / 200, false
}
