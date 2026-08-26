package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func testApp(t *testing.T) App {
	t.Helper()
	a, err := NewApp("")
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = 80, 24
	return a
}

func press(a App, msg tea.KeyMsg) (App, tea.Cmd) {
	m, cmd := a.Update(msg)
	return m.(App), cmd
}

func typeRunes(a App, s string) App {
	for _, r := range s {
		a, _ = press(a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return a
}

// A quit usually travels batched with the window-title command, so look inside.
func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	switch msg := cmd().(type) {
	case tea.QuitMsg:
		return true
	case tea.BatchMsg:
		for _, c := range msg {
			if isQuit(c) {
				return true
			}
		}
	}
	return false
}

func TestQuitAsksOnlyWhenThereIsSomethingToLose(t *testing.T) {
	t.Run("a clean document quits", func(t *testing.T) {
		_, cmd := press(testApp(t), tea.KeyMsg{Type: tea.KeyCtrlQ})

		if !isQuit(cmd) {
			t.Error("ctrl+q on a clean document did not quit")
		}
	})

	t.Run("an edited document asks first", func(t *testing.T) {
		a := typeRunes(testApp(t), "hola")

		a, cmd := press(a, tea.KeyMsg{Type: tea.KeyCtrlQ})

		if a.mode != ModeConfirm {
			t.Errorf("mode = %v, want ModeConfirm", a.mode)
		}
		if isQuit(cmd) {
			t.Error("ctrl+q quit without waiting for the answer")
		}
	})
}

// Opening a file replaces the buffer. Every way of losing unsaved work has to
// ask first, and this is one of them.
func TestOpenAsksBeforeDiscardingChanges(t *testing.T) {
	a := typeRunes(testApp(t), "un borrador")

	a, _ = press(a, tea.KeyMsg{Type: tea.KeyCtrlO})

	if a.mode != ModeConfirm {
		t.Fatalf("mode = %v, want ModeConfirm", a.mode)
	}
	if got := a.ed.Text(); got != "un borrador" {
		t.Errorf("the draft was touched: %q", got)
	}
}

func TestAnsweringNoKeepsTheDraft(t *testing.T) {
	a := typeRunes(testApp(t), "hola")
	a, _ = press(a, tea.KeyMsg{Type: tea.KeyCtrlO})

	a, _ = press(a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if a.mode != ModeWrite {
		t.Errorf("mode = %v, want ModeWrite", a.mode)
	}
	if got := a.ed.Text(); got != "hola" {
		t.Errorf("document = %q, want it untouched", got)
	}
}

func TestAnsweringYesRunsTheAction(t *testing.T) {
	a := typeRunes(testApp(t), "hola")
	a, _ = press(a, tea.KeyMsg{Type: tea.KeyCtrlN})

	a, _ = press(a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if got := a.ed.Text(); got != "" {
		t.Errorf("document = %q, want a blank one", got)
	}
}

func TestHelpAndAbout(t *testing.T) {
	a, _ := press(testApp(t), tea.KeyMsg{Type: tea.KeyF1})
	if a.mode != ModeHelp {
		t.Fatalf("f1 gave mode %v, want ModeHelp", a.mode)
	}

	a, _ = press(a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if a.mode != ModeAbout {
		t.Errorf("? in help gave mode %v, want ModeAbout", a.mode)
	}

	a, _ = press(a, tea.KeyMsg{Type: tea.KeyEsc})
	if a.mode != ModeWrite {
		t.Errorf("esc gave mode %v, want ModeWrite", a.mode)
	}
}

// The session and document word goals are the settings justwrite has, and
// they are changed from inside the stats panel rather than by hand-editing a
// file.
func TestSetSessionGoalFromStatsPanel(t *testing.T) {
	withTempConfigDir(t)
	a, _ := press(testApp(t), tea.KeyMsg{Type: tea.KeyCtrlT})
	if a.mode != ModeStats {
		t.Fatalf("ctrl+t gave mode %v, want ModeStats", a.mode)
	}

	a, _ = press(a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if a.editingGoal != sessionGoalField {
		t.Fatal("g did not start editing the session goal")
	}

	a = typeRunes(a, "500")
	a, _ = press(a, tea.KeyMsg{Type: tea.KeyEnter})

	if a.editingGoal != noGoalField {
		t.Error("enter did not stop editing")
	}
	if a.cfg.SessionGoal != 500 {
		t.Errorf("cfg.SessionGoal = %d, want 500", a.cfg.SessionGoal)
	}
	if c := loadConfig(); c.SessionGoal != 500 {
		t.Errorf("persisted SessionGoal = %d, want 500 — the point is it survives a restart", c.SessionGoal)
	}
}

// The document goal is a separate setting from the session goal — a book's
// total length target rather than a single sitting's.
func TestSetDocGoalFromStatsPanel(t *testing.T) {
	withTempConfigDir(t)
	a, _ := press(testApp(t), tea.KeyMsg{Type: tea.KeyCtrlT})

	a, _ = press(a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if a.editingGoal != docGoalField {
		t.Fatal("d did not start editing the document goal")
	}

	a = typeRunes(a, "50000")
	a, _ = press(a, tea.KeyMsg{Type: tea.KeyEnter})

	if a.cfg.DocGoal != 50000 {
		t.Errorf("cfg.DocGoal = %d, want 50000", a.cfg.DocGoal)
	}
	if a.cfg.SessionGoal != 0 {
		t.Errorf("cfg.SessionGoal = %d, want 0 — setting the doc goal must not touch it", a.cfg.SessionGoal)
	}
	if c := loadConfig(); c.DocGoal != 50000 {
		t.Errorf("persisted DocGoal = %d, want 50000", c.DocGoal)
	}
}

func TestEscCancelsGoalEditingWithoutChanging(t *testing.T) {
	withTempConfigDir(t)
	a := testApp(t)
	a.cfg.SessionGoal = 200

	a, _ = press(a, tea.KeyMsg{Type: tea.KeyCtrlT})
	a, _ = press(a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	a = typeRunes(a, "999")
	a, _ = press(a, tea.KeyMsg{Type: tea.KeyEsc})

	if a.editingGoal != noGoalField {
		t.Error("esc did not stop editing")
	}
	if a.cfg.SessionGoal != 200 {
		t.Errorf("cfg.SessionGoal = %d, want unchanged 200", a.cfg.SessionGoal)
	}
}

// An empty field clears the goal rather than being rejected as invalid — not
// wanting a target anymore is a valid choice.
func TestClearingTheGoalInputSetsNoGoal(t *testing.T) {
	withTempConfigDir(t)
	a := testApp(t)
	a.cfg.SessionGoal = 500

	a, _ = press(a, tea.KeyMsg{Type: tea.KeyCtrlT})
	a, _ = press(a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	for range 3 { // clear the prefilled "500"
		a, _ = press(a, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	a, _ = press(a, tea.KeyMsg{Type: tea.KeyEnter})

	if a.cfg.SessionGoal != 0 {
		t.Errorf("cfg.SessionGoal = %d, want 0 (cleared)", a.cfg.SessionGoal)
	}
}

// The shortcut list no longer fits in one screenful, so it scrolls — but
// never past either end.
func TestHelpScrolls(t *testing.T) {
	a, _ := press(testApp(t), tea.KeyMsg{Type: tea.KeyF1})

	wantMax := helpMaxScroll(a.helpVisibleBands())

	for range helpBandCount() + 5 {
		a, _ = press(a, tea.KeyMsg{Type: tea.KeyDown})
	}
	if a.helpScroll != wantMax {
		t.Errorf("helpScroll = %d after scrolling well past the end, want %d (clamped)", a.helpScroll, wantMax)
	}

	a, _ = press(a, tea.KeyMsg{Type: tea.KeyPgUp})
	if a.helpScroll != 0 {
		t.Errorf("helpScroll = %d after paging up from the end, want 0", a.helpScroll)
	}

	for range helpBandCount() + 5 {
		a, _ = press(a, tea.KeyMsg{Type: tea.KeyUp})
	}
	if a.helpScroll != 0 {
		t.Errorf("helpScroll = %d after scrolling well past the start, want 0 (clamped)", a.helpScroll)
	}
}

// ? is a character before it is a shortcut: this is an editor.
func TestQuestionMarkTypesWhileWriting(t *testing.T) {
	a := typeRunes(testApp(t), "¿qué?")

	if a.mode != ModeWrite {
		t.Errorf("mode = %v, want to still be writing", a.mode)
	}
	if got := a.ed.Text(); got != "¿qué?" {
		t.Errorf("document = %q", got)
	}
}

// A save that fails without saying so is a draft lost quietly.
func TestAFailedSaveShowsUpOnTheBar(t *testing.T) {
	a := testApp(t)
	a.ed.Path = t.TempDir() // a directory can never be written as a file
	a = typeRunes(a, "hola")

	a, _ = press(a, tea.KeyMsg{Type: tea.KeyCtrlS})

	if !a.statusErr || a.status == "" {
		t.Fatalf("statusErr=%v status=%q, want the reason on the bar", a.statusErr, a.status)
	}
	if !a.ed.Modified {
		t.Error("the document was marked saved even though the save failed")
	}
}

func TestSavingAKnownFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nota.md")
	a := testApp(t)
	a.ed.Path = path
	a = typeRunes(a, "hola")

	a, _ = press(a, tea.KeyMsg{Type: tea.KeyCtrlS})

	if a.statusErr {
		t.Fatalf("save reported an error: %q", a.status)
	}
	if b, err := os.ReadFile(path); err != nil || string(b) != "hola" {
		t.Errorf("file = %q (%v)", b, err)
	}
	if a.status != "saved" {
		t.Errorf("status = %q, want %q", a.status, "saved")
	}
}

// A document with no name asks for one instead of failing quietly.
func TestSavingAnUnnamedDocumentOpensTheDialog(t *testing.T) {
	a := typeRunes(testApp(t), "hola")

	a, _ = press(a, tea.KeyMsg{Type: tea.KeyCtrlS})

	if a.mode != ModeSaveAs {
		t.Errorf("mode = %v, want ModeSaveAs", a.mode)
	}
	if !a.onName {
		t.Error("focus should start in the filename field, which is what you pressed save to fill in")
	}
}

// browse scrolled by the full panel height, but save-as only draws
// height-2 rows for the listing — the other two hold the filename field —
// so holding the cursor down could push it below what filePanel renders.
func TestSaveAsScrollStaysInsideWhatItDraws(t *testing.T) {
	dir := t.TempDir()
	for i := range 40 {
		name := filepath.Join(dir, fmt.Sprintf("archivo%02d.md", i))
		if err := os.WriteFile(name, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	a := testApp(t)
	a.mode = ModeSaveAs
	a.exp.refresh(dir, "")

	for range 50 {
		a.browse(tea.KeyMsg{Type: tea.KeyDown})
	}

	rows := a.listRows()
	if a.exp.cursor < a.exp.offset || a.exp.cursor >= a.exp.offset+rows {
		t.Errorf("cursor %d outside the visible window [%d, %d)", a.exp.cursor, a.exp.offset, a.exp.offset+rows)
	}
}

// commitName compared an absolute path built from a.exp.dir against
// a.ed.Path exactly as the user typed it — relative if justwrite was
// launched as `justwrite nota.md`. Saving over the file already open under
// that name must not ask to overwrite itself.
func TestSaveAsOverTheOpenFileAsksNothing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nota.md"), []byte("hola"), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	a, err := NewApp("nota.md")
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	a.exp.refresh(a.startDir(), "")
	a.name.SetValue("nota.md")

	a.commitName()

	if a.mode == ModeConfirm {
		t.Error("asked to overwrite the file that is already open")
	}
}

// Home, End and PageUp all index into the rows of the page, so a terminal too
// narrow to hold any text still has to produce one. If that guard ever goes,
// this test panics rather than fails.
func TestANarrowTerminalDoesNotPanic(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {5, 3}, {12, 10}, {29, 9}, {30, 10}, {80, 24}} {
		a := testApp(t)
		a.w, a.h = size[0], size[1]
		a = typeRunes(a, "hola mundo, un texto que no cabe")

		for _, k := range []tea.KeyType{
			tea.KeyHome, tea.KeyEnd, tea.KeyPgUp, tea.KeyPgDown,
			tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight,
		} {
			a, _ = press(a, tea.KeyMsg{Type: k})
		}
		_ = a.View()
	}
}

// A document is modified when its text changes, and at no other time. Keys
// that write nothing must not put a * on the bar — on Windows a bare Shift or
// Ctrl arrives as a key event carrying a NUL rune, and an alt chord is a
// command, not a character.
func TestKeysThatWriteNothingDoNotModifyTheDocument(t *testing.T) {
	for _, c := range []struct {
		name string
		msg  tea.KeyMsg
	}{
		{"a bare modifier", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}}},
		{"an alt chord", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}, Alt: true}},
		{"an unbound function key", tea.KeyMsg{Type: tea.KeyF5}},
		{"an unbound ctrl chord", tea.KeyMsg{Type: tea.KeyCtrlW}},
		{"help", tea.KeyMsg{Type: tea.KeyF1}},
		{"a cursor move", tea.KeyMsg{Type: tea.KeyRight}},
		{"select all on an empty document", tea.KeyMsg{Type: tea.KeyCtrlA}},
	} {
		t.Run(c.name, func(t *testing.T) {
			a, _ := press(testApp(t), c.msg)

			if a.ed.Modified {
				t.Errorf("the document is marked modified after %q", c.msg.String())
			}
			if got := a.ed.Text(); got != "" {
				t.Errorf("the document gained %q", got)
			}
		})
	}
}

func TestTypingIsNotSwallowedByPanels(t *testing.T) {
	a, _ := press(testApp(t), tea.KeyMsg{Type: tea.KeyCtrlT})

	a = typeRunes(a, "hola")

	if got := a.ed.Text(); got != "" {
		t.Errorf("keys reached the document through the stats panel: %q", got)
	}
}
