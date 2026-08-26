package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/MawCeron/justwrite/internal/editor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

type Mode int

const (
	ModeWrite Mode = iota
	ModeOpen
	ModeSaveAs
	ModeStats
	ModeHelp
	ModeAbout
	ModeConfirm
	ModeConflict // the file changed on disk since it was opened; ctrl+s stopped
	ModeFind
)

// confirmation is a question whose yes answer throws something away.
type confirmation struct {
	message string
	yes     func(*App) tea.Cmd
}

// clearStatus retires a transient message, identified by the sequence number it
// was posted with so a stale timer cannot wipe a newer message.
type clearStatus int

const statusLinger = 2 * time.Second

type App struct {
	ed   *editor.Editor
	mode Mode
	w, h int

	status    string
	statusErr bool
	statusSeq int

	confirm confirmation

	exp    explorer
	name   textinput.Model
	onName bool // save-as: the filename field has the focus, not the listing

	helpScroll int // how many shortcut cards have scrolled off the top

	cfg         config
	goalInput   textinput.Model
	editingGoal goalField // stats: which goal field has the focus, if any

	findInput   textinput.Model
	onFindQuery bool // find: typing a query, rather than stepping through matches
	findOrigin  int  // cursor position from before the search, restored on esc

	title string // last title handed to the terminal
}

func NewApp(path string) (App, error) {
	ed := editor.New()
	if path != "" {
		if err := ed.Load(path); err != nil {
			return App{}, err
		}
	}

	name := textinput.New()
	name.Prompt = ""
	name.Placeholder = "filename.md"
	name.TextStyle = overlayStyle
	name.PlaceholderStyle = overlayDimStyle
	name.Cursor.Style = cursorStyle

	goalInput := textinput.New()
	goalInput.Prompt = ""
	goalInput.Placeholder = "500"
	goalInput.CharLimit = 6
	goalInput.TextStyle = overlayStyle
	goalInput.PlaceholderStyle = overlayDimStyle
	goalInput.Cursor.Style = cursorStyle

	findInput := textinput.New()
	findInput.Prompt = ""
	findInput.Placeholder = "search"
	findInput.TextStyle = overlayStyle
	findInput.PlaceholderStyle = overlayDimStyle
	findInput.Cursor.Style = cursorStyle

	// A sane size until the terminal reports its own.
	return App{ed: ed, name: name, goalInput: goalInput, findInput: findInput, cfg: loadConfig(), w: 80, h: 24}, nil
}

func (a App) Init() tea.Cmd { return nil }

// Update dispatches the message and then, if the document's identity changed,
// tells the terminal about it — rather than repainting the title every frame.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := a.update(msg)
	if t := next.windowTitle(); t != next.title {
		next.title = t
		cmd = tea.Batch(cmd, tea.SetWindowTitle(t))
	}
	return next, cmd
}

func (a App) update(msg tea.Msg) (App, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height

	case clearStatus:
		// Errors stay put until something goes right; a save that failed is
		// not something to notice or miss within two seconds.
		if int(msg) == a.statusSeq && !a.statusErr {
			a.status = ""
		}

	case tea.KeyMsg:
		switch a.mode {
		case ModeWrite:
			return a.keyWrite(msg)
		case ModeOpen:
			return a.keyOpen(msg)
		case ModeSaveAs:
			return a.keySaveAs(msg)
		case ModeConfirm:
			return a.keyConfirm(msg)
		case ModeHelp:
			return a.keyHelp(msg)
		case ModeStats:
			return a.keyStats(msg)
		case ModeAbout:
			return a.keyClosePanel(msg)
		case ModeConflict:
			return a.keyConflict(msg)
		case ModeFind:
			return a.keyFind(msg)
		}

	default:
		// Everything that is not a key press or one of the two above — the
		// cursor's own blink tick, chiefly — still has to reach whichever
		// text field is currently focused, or its cursor stops blinking and
		// can end up stuck invisible.
		var cmd tea.Cmd
		switch {
		case a.mode == ModeSaveAs && a.onName:
			a.name, cmd = a.name.Update(msg)
		case a.mode == ModeStats && a.editingGoal != noGoalField:
			a.goalInput, cmd = a.goalInput.Update(msg)
		case a.mode == ModeFind && a.onFindQuery:
			a.findInput, cmd = a.findInput.Update(msg)
		}
		return a, cmd
	}
	return a, nil
}

func (a App) windowTitle() string {
	name := "untitled"
	if a.ed.Path != "" {
		name = filepath.Base(a.ed.Path)
	}
	if a.ed.Modified {
		name += "*"
	}
	return name + " — " + appName
}

// ─── Writing ─────────────────────────────────────────────────────────────────

func (a App) keyWrite(msg tea.KeyMsg) (App, tea.Cmd) {
	vlines := a.ed.VisualLines(a.textWidth())
	page := a.textHeight()

	switch msg.String() {
	// ── File ──
	case "ctrl+q":
		return a.confirmed("quit without saving?", func(a *App) tea.Cmd { return tea.Quit })
	case "ctrl+n":
		return a.confirmed("discard changes?", func(a *App) tea.Cmd { a.ed.NewDocument(); return nil })
	case "ctrl+o":
		// Opening replaces the buffer, which is one more way to lose a draft.
		return a.confirmed("discard changes?", (*App).openDialog)
	case "ctrl+s":
		return a, a.saveNow()

	// ── Edit ──
	case "ctrl+z":
		a.ed.Undo()
	case "ctrl+y":
		a.ed.Redo()
	case "ctrl+a":
		a.ed.SelectAll()
	case "ctrl+c":
		if s := a.ed.Copy(); s != "" {
			return a, toClipboard(s)
		}
	case "ctrl+x":
		if s := a.ed.Cut(); s != "" {
			return a, toClipboard(s)
		}
	case "ctrl+v":
		a.ed.Paste("")

	// ── Panels ──
	case "ctrl+f":
		return a, a.openFind()
	case "ctrl+t":
		a.mode = ModeStats
	case "f1":
		a.mode = ModeHelp
		a.helpScroll = 0

	// ── Navigation ──
	case "ctrl+left":
		a.ed.WordLeft(false)
	case "ctrl+right":
		a.ed.WordRight(false)
	case "ctrl+shift+left":
		a.ed.WordLeft(true)
	case "ctrl+shift+right":
		a.ed.WordRight(true)
	case "left":
		a.ed.MoveLeft(false)
	case "shift+left":
		a.ed.MoveLeft(true)
	case "right":
		a.ed.MoveRight(false)
	case "shift+right":
		a.ed.MoveRight(true)
	case "up":
		a.ed.MoveUp(vlines, false)
	case "shift+up":
		a.ed.MoveUp(vlines, true)
	case "down":
		a.ed.MoveDown(vlines, false)
	case "shift+down":
		a.ed.MoveDown(vlines, true)
	case "home":
		a.ed.Home(vlines, false)
	case "shift+home":
		a.ed.Home(vlines, true)
	case "end":
		a.ed.End(vlines, false)
	case "shift+end":
		a.ed.End(vlines, true)
	case "ctrl+home":
		a.ed.DocumentHome(false)
	case "ctrl+shift+home":
		a.ed.DocumentHome(true)
	case "ctrl+end":
		a.ed.DocumentEnd(false)
	case "ctrl+shift+end":
		a.ed.DocumentEnd(true)
	case "pgup":
		a.ed.PageUp(vlines, page, false)
	case "shift+pgup":
		a.ed.PageUp(vlines, page, true)
	case "pgdown":
		a.ed.PageDown(vlines, page, false)
	case "shift+pgdown":
		a.ed.PageDown(vlines, page, true)

	// ── Input ──
	case "enter":
		a.ed.InsertRune('\n')
	case "backspace":
		a.ed.Backspace()
	case "ctrl+w":
		a.ed.DeleteWordLeft()
	case "delete":
		a.ed.DeleteForward()
	case "tab":
		a.ed.InsertString("    ")
	default:
		a.typeInto(msg)
	}
	return a, nil
}

// toClipboard hands the text to the terminal emulator with an OSC 52 escape.
// The emulator is the one that owns the clipboard, so this works over ssh and
// inside tmux — where a system-clipboard library needs a display server the
// writerdeck does not have.
//
// ponytail: writes to stdout beside the renderer, so a copy can land mid-frame
// and flicker until the next repaint. Harmless; needs the renderer's lock to
// fix, which v1 does not expose.
func toClipboard(s string) tea.Cmd {
	return func() tea.Msg {
		termenv.Copy(s)
		return nil
	}
}

// typeInto puts ordinary keystrokes into the document. This is the only way
// text gets in from the keyboard, so it is the one place that decides what
// counts as typing — and nothing that is not typing may mark the document
// modified.
//
// A bracketed paste arrives the same way, but lands as one undo step rather
// than a hundred.
func (a *App) typeInto(msg tea.KeyMsg) {
	if msg.Paste {
		a.ed.Paste(string(msg.Runes))
		return
	}
	for _, r := range typedRunes(msg) {
		a.ed.InsertRune(r)
	}
}

// typedRunes returns the characters a key event actually types, and nothing
// for a key event that is a command instead. Both the document and the file
// filter go through here, so "what counts as typing" is decided once.
func typedRunes(msg tea.KeyMsg) string {
	if msg.Type == tea.KeySpace {
		return " "
	}
	// An alt chord is a command we have no binding for, not a character.
	if msg.Type != tea.KeyRunes || msg.Alt {
		return ""
	}

	out := make([]rune, 0, len(msg.Runes))
	for _, r := range msg.Runes {
		// Windows reports a bare Shift, Ctrl or Caps Lock as a key event
		// carrying a NUL. Newline and tab have keys of their own, so nothing
		// printable is lost by refusing control runes here.
		if !unicode.IsControl(r) {
			out = append(out, r)
		}
	}
	return string(out)
}

// confirmed runs action, asking first when there is unsaved work to lose.
func (a App) confirmed(message string, action func(*App) tea.Cmd) (App, tea.Cmd) {
	if !a.ed.Modified {
		return a, action(&a)
	}
	a.confirm = confirmation{message: message, yes: action}
	a.mode = ModeConfirm
	return a, nil
}

// ─── Saving ──────────────────────────────────────────────────────────────────

func (a *App) saveNow() tea.Cmd {
	if a.ed.Path == "" {
		return a.saveAsDialog()
	}
	if err := a.ed.Save(); err != nil {
		if errors.Is(err, editor.ErrExternalChange) {
			a.mode = ModeConflict
			return nil
		}
		return a.failed(err)
	}
	return a.posted("saved")
}

func (a *App) saveTo(path string) tea.Cmd {
	if err := a.ed.SaveAs(path); err != nil {
		return a.failed(err)
	}
	a.mode = ModeWrite
	return a.posted("saved")
}

// failed puts the reason on the bar. A save that goes wrong silently is a
// draft lost without anybody noticing.
func (a *App) failed(err error) tea.Cmd {
	a.status, a.statusErr = err.Error(), true
	a.statusSeq++
	return nil
}

func (a *App) posted(text string) tea.Cmd {
	a.status, a.statusErr = text, false
	a.statusSeq++
	seq := a.statusSeq
	return tea.Tick(statusLinger, func(time.Time) tea.Msg { return clearStatus(seq) })
}

// ─── Dialogs ─────────────────────────────────────────────────────────────────

// startDir is where a file dialog opens: beside the current document, or the
// working directory for one that has no home yet.
func (a App) startDir() string {
	if a.ed.Path != "" {
		if dir := filepath.Dir(a.ed.Path); dir != "" {
			return dir
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

func (a *App) openDialog() tea.Cmd {
	a.exp.refresh(a.startDir(), "")
	a.mode = ModeOpen
	return nil
}

func (a *App) saveAsDialog() tea.Cmd {
	a.exp.refresh(a.startDir(), "")
	suggested := ""
	if a.ed.Path != "" {
		suggested = filepath.Base(a.ed.Path)
	}
	a.name.SetValue(suggested)
	a.name.CursorEnd()
	a.onName = true // you pressed save to name the thing, so start in the field
	a.mode = ModeSaveAs
	return a.name.Focus()
}

// openFind starts a search, remembering where the cursor was so esc can put
// it back — the search itself may jump it all over the document.
func (a *App) openFind() tea.Cmd {
	a.findOrigin = a.ed.Cursor()
	a.findInput.SetValue("")
	a.findInput.CursorEnd()
	a.onFindQuery = true
	a.mode = ModeFind
	return a.findInput.Focus()
}

func (a App) keyOpen(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Esc backs out one step at a time: first the filter, then the dialog.
		if a.exp.filter != "" {
			a.exp.setFilter("")
			return a, nil
		}
		a.mode = ModeWrite
	case "enter":
		return a.enterSelection(false)
	default:
		a.browse(msg)
	}
	return a, nil
}

func (a App) keySaveAs(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.name.Blur()
		a.mode = ModeWrite
		return a, nil
	case "tab", "shift+tab":
		a.onName = !a.onName
		if a.onName {
			return a, a.name.Focus()
		}
		a.name.Blur()
		return a, nil
	}

	if !a.onName {
		if msg.String() == "enter" {
			return a.enterSelection(true)
		}
		a.browse(msg)
		return a, nil
	}

	if msg.String() == "enter" {
		return a, a.commitName()
	}
	var cmd tea.Cmd
	a.name, cmd = a.name.Update(msg)
	return a, cmd
}

// browse handles the keys the listing shares between both dialogs.
func (a *App) browse(msg tea.KeyMsg) {
	window := a.listRows()
	switch msg.String() {
	case "up":
		a.exp.move(-1, window)
	case "down":
		a.exp.move(1, window)
	case "pgup":
		a.exp.move(-window, window)
	case "pgdown":
		a.exp.move(window, window)
	case "ctrl+h":
		a.exp.toggleAll()
	case "backspace":
		if r := []rune(a.exp.filter); len(r) > 0 {
			a.exp.setFilter(string(r[:len(r)-1]))
		}
	default:
		if s := typedRunes(msg); s != "" && !msg.Paste {
			a.exp.setFilter(a.exp.filter + s)
		}
	}
}

// enterSelection descends into a directory, or acts on a file: opening it, or
// borrowing its name when the point is to save over it.
func (a App) enterSelection(naming bool) (App, tea.Cmd) {
	it, ok := a.exp.selected()
	if !ok {
		return a, nil
	}

	if it.isDir {
		focus := ""
		if it.isUp {
			focus = filepath.Base(a.exp.dir) // land on the directory just left
		}
		a.exp.refresh(it.path, focus)
		return a, nil
	}

	if naming {
		a.name.SetValue(it.name)
		a.name.CursorEnd()
		a.onName = true
		return a, a.name.Focus()
	}

	if err := a.ed.Load(it.path); err != nil {
		return a, a.failed(err)
	}
	a.mode = ModeWrite
	return a, nil
}

// commitName saves under the typed name, asking before it writes over
// something that is already there.
func (a *App) commitName() tea.Cmd {
	name := strings.TrimSpace(a.name.Value())
	if name == "" {
		return nil
	}
	path := filepath.Join(a.exp.dir, name)

	edPath := a.ed.Path
	if abs, err := filepath.Abs(edPath); err == nil {
		edPath = abs
	}

	if _, err := os.Stat(path); err == nil && path != edPath {
		a.name.Blur()
		a.confirm = confirmation{
			message: "overwrite " + filepath.Base(path) + "?",
			yes:     func(a *App) tea.Cmd { return a.saveTo(path) },
		}
		a.mode = ModeConfirm
		return nil
	}
	a.name.Blur()
	return a.saveTo(path)
}

// ─── Panels ──────────────────────────────────────────────────────────────────

func (a App) keyConfirm(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		action := a.confirm.yes
		a.confirm = confirmation{}
		a.mode = ModeWrite // the action may choose another mode for itself
		if action == nil {
			return a, nil
		}
		return a, action(&a)
	case "n", "N", "esc":
		a.confirm = confirmation{}
		a.mode = ModeWrite
	}
	return a, nil
}

// keyConflict handles the three ways out of ErrExternalChange: take the copy
// on disk, write over it anyway, or keep both by naming this one differently.
func (a App) keyConflict(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "r":
		a.mode = ModeWrite
		if err := a.ed.Load(a.ed.Path); err != nil {
			return a, a.failed(err)
		}
		return a, a.posted("reloaded")
	case "o":
		a.mode = ModeWrite
		if err := a.ed.ForceSave(); err != nil {
			return a, a.failed(err)
		}
		return a, a.posted("saved")
	case "s":
		a.mode = ModeWrite
		return a, a.saveAsDialog()
	case "esc":
		a.mode = ModeWrite
	}
	return a, nil
}

// keyFind handles ctrl+f: typing the query field until the first enter, then
// enter/n and N step through matches instead of typing into it — the same
// two-phase shape a shell's reverse search uses, so "n" and "N" are free for
// navigation once there is something to navigate.
func (a App) keyFind(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.ed.SetCursor(a.findOrigin)
		a.findInput.Blur()
		a.onFindQuery = false
		a.mode = ModeWrite
		return a, nil
	case "tab":
		// Back to the field to change the query, without closing and
		// reopening find just to search for something else.
		a.onFindQuery = !a.onFindQuery
		if a.onFindQuery {
			a.findInput.CursorEnd()
			return a, a.findInput.Focus()
		}
		a.findInput.Blur()
		return a, nil
	}

	if a.onFindQuery {
		if msg.String() == "enter" {
			if !a.ed.Find(a.findInput.Value(), false) {
				return a, a.posted("no match")
			}
			a.onFindQuery = false
			a.findInput.Blur()
			return a, nil
		}
		var cmd tea.Cmd
		a.findInput, cmd = a.findInput.Update(msg)
		return a, cmd
	}

	switch msg.String() {
	case "enter", "n":
		if !a.ed.Find(a.findInput.Value(), false) {
			return a, a.posted("no match")
		}
	case "N":
		if !a.ed.Find(a.findInput.Value(), true) {
			return a, a.posted("no match")
		}
	}
	return a, nil
}

func (a App) keyHelp(msg tea.KeyMsg) (App, tea.Cmd) {
	visible := a.helpVisibleBands()
	switch msg.String() {
	case "?":
		// Only here: while writing, ? is a question mark like any other.
		a.mode = ModeAbout
	case "f1", "esc", "enter", "q":
		a.mode = ModeWrite
	case "up":
		a.helpScroll = max(a.helpScroll-1, 0)
	case "down":
		a.helpScroll = min(a.helpScroll+1, helpMaxScroll(visible))
	case "pgup":
		a.helpScroll = max(a.helpScroll-visible, 0)
	case "pgdown":
		a.helpScroll = min(a.helpScroll+visible, helpMaxScroll(visible))
	}
	return a, nil
}

func (a App) keyClosePanel(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q", "f1", "ctrl+t", "?":
		a.mode = ModeWrite
	}
	return a, nil
}

// goalField is which of the two goals the stats panel's input row is
// currently editing, if either.
type goalField int

const (
	noGoalField goalField = iota
	sessionGoalField
	docGoalField
)

// parseGoalInput reads the typed number: empty means "clear the goal", a
// valid non-negative integer becomes the new one, and anything else — bad
// input — leaves the previous value alone rather than zeroing it out.
func parseGoalInput(v string) (n int, ok bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, true
	}
	n, err := strconv.Atoi(v)
	return n, err == nil && n >= 0
}

// keyStats handles the stats panel, including setting the session and
// document word goals — the settings justwrite has, changed from inside the
// app rather than by hand-editing a file.
func (a App) keyStats(msg tea.KeyMsg) (App, tea.Cmd) {
	if a.editingGoal != noGoalField {
		switch msg.String() {
		case "enter":
			if n, ok := parseGoalInput(a.goalInput.Value()); ok {
				switch a.editingGoal {
				case sessionGoalField:
					a.cfg.SessionGoal = n
				case docGoalField:
					a.cfg.DocGoal = n
				}
				a.cfg.save()
			}
			a.goalInput.Blur()
			a.editingGoal = noGoalField
		case "esc":
			a.goalInput.Blur()
			a.editingGoal = noGoalField
		default:
			var cmd tea.Cmd
			a.goalInput, cmd = a.goalInput.Update(msg)
			return a, cmd
		}
		return a, nil
	}

	startEditing := func(field goalField, current int) (App, tea.Cmd) {
		a.goalInput.SetValue("")
		if current > 0 {
			a.goalInput.SetValue(strconv.Itoa(current))
		}
		a.goalInput.CursorEnd()
		a.editingGoal = field
		return a, a.goalInput.Focus()
	}

	switch msg.String() {
	case "g":
		return startEditing(sessionGoalField, a.cfg.SessionGoal)
	case "d":
		return startEditing(docGoalField, a.cfg.DocGoal)
	}
	return a.keyClosePanel(msg)
}
