package editor

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestSaveWritesTheDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nota.md")
	e := New()
	e.SetText("hola")
	e.Path = path
	e.Modified = true

	if err := e.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if b, err := os.ReadFile(path); err != nil || string(b) != "hola" {
		t.Errorf("file is %q (%v), want %q", b, err, "hola")
	}
	if e.Modified {
		t.Error("the document is still marked modified after a good save")
	}
	// The write goes through a temporary file; leaving it behind would litter
	// the writing directory with stray files on every ctrl+s.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("directory has %d entries after save, want 1 (the temp file was left behind)", len(entries))
	}
}

// apply() used to mark the document modified unconditionally, so undoing
// back to exactly what is on disk still showed the * and still asked
// "discard changes?" on quit.
func TestUndoBackToSavedStateClearsModified(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nota.md")
	e := New()
	e.SetText("hola")
	e.Path = path
	if err := e.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	e.InsertRune('!')
	if !e.Modified {
		t.Fatal("typing did not mark the document modified")
	}

	e.Undo()

	if e.Modified {
		t.Error("undoing back to the saved text left the modified marker on")
	}
	if got := e.Text(); got != "hola" {
		t.Errorf("Text() = %q, want %q", got, "hola")
	}
}

// A diverged history can return to the same undo depth as the save without
// returning to the same content: undo twice, then start a different edit —
// undoing that lands one op deep either way. Comparing which edit is on top,
// not how many are, is what keeps that from reading as "not modified."
func TestDivergedHistoryAtTheSameDepthIsStillModified(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nota.md")
	e := New()
	e.InsertString("a")
	e.Path = path
	if err := e.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	e.InsertString("b") // depth 2: "ab"
	e.Undo()            // depth 1: "a" — back to the saved text
	if e.Modified {
		t.Fatal("undoing back to the saved text should have cleared Modified")
	}

	e.Undo()            // depth 0: ""
	e.InsertString("c") // a new branch, also depth 1, but "c" was never saved
	e.InsertString("d") // depth 2: "cd"
	e.Undo()            // depth 1 again: "c"

	if got := e.Text(); got != "c" {
		t.Fatalf("Text() = %q, want %q (test setup check)", got, "c")
	}
	if !e.Modified {
		t.Error("same undo depth as the save on a different branch, but text is \"c\" not \"a\" — should still be modified")
	}
}

// Save() draws its temporary name from os.CreateTemp, so a file that happens
// to already be named <doc>.tmp must not be mistaken for it and destroyed.
func TestSaveDoesNotClobberAPreexistingTmpFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nota.md")
	foreign := path + ".tmp"
	if err := os.WriteFile(foreign, []byte("no me toques"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New()
	e.SetText("hola")
	e.Path = path
	if err := e.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if b, err := os.ReadFile(foreign); err != nil || string(b) != "no me toques" {
		t.Errorf("the pre-existing .tmp file is %q (%v), want it untouched", b, err)
	}
}

// A document opened at a stricter mode than 0644 — a private journal at 0600,
// say — must not come back looser just because it went through Save().
func TestSavePreservesFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows file modes don't distinguish 0600 from 0644")
	}

	path := filepath.Join(t.TempDir(), "privado.md")
	if err := os.WriteFile(path, []byte("secreto"), 0o600); err != nil {
		t.Fatal(err)
	}

	e := New()
	if err := e.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	e.InsertString(" nuevo")
	if err := e.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode = %o, want 0600", got)
	}
}

// Save must not clobber a file that changed on disk after it was opened —
// another editor, a sync client, another justwrite instance.
func TestSaveRefusesAfterAnExternalChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nota.md")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New()
	if err := e.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	e.InsertString(" editado")

	// Something else touched the file after it was opened here. Chtimes
	// rather than a second WriteFile: a filesystem's mtime resolution can be
	// coarser than how fast two writes in a test run apart.
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}

	if err := e.Save(); !errors.Is(err, ErrExternalChange) {
		t.Fatalf("Save() = %v, want ErrExternalChange", err)
	}
	if b, _ := os.ReadFile(path); string(b) != "original" {
		t.Errorf("the file on disk changed to %q, want it left alone", b)
	}
}

// The ordinary path — open a file, edit it, save it, nothing else touches it
// — must not be affected by the guard against everything else.
func TestSaveSucceedsWithoutAnExternalChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nota.md")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New()
	if err := e.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	e.InsertString(" editado")

	if err := e.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if b, _ := os.ReadFile(path); string(b) != "original editado" {
		t.Errorf("file = %q, want %q", b, "original editado")
	}
}

// ForceSave is the "overwrite anyway" choice: it writes through the conflict
// Save just refused.
func TestForceSaveOverwritesDespiteAnExternalChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nota.md")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New()
	if err := e.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	e.InsertString(" editado")

	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}

	if err := e.ForceSave(); err != nil {
		t.Fatalf("ForceSave: %v", err)
	}
	if b, _ := os.ReadFile(path); string(b) != "original editado" {
		t.Errorf("file = %q, want %q", b, "original editado")
	}

	// The just-written state becomes the new baseline — saving again right
	// after must not immediately refuse itself.
	if err := e.Save(); err != nil {
		t.Errorf("Save() after ForceSave = %v, want nil", err)
	}
}

// Reloading after a conflict picks up the change and clears it — Save
// works normally again once the document matches what is on disk.
func TestReloadClearsTheConflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nota.md")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New()
	if err := e.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}

	future := time.Now().Add(time.Hour)
	if err := os.WriteFile(path, []byte("changed elsewhere"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}

	if err := e.Load(path); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := e.Text(); got != "changed elsewhere" {
		t.Fatalf("Text() = %q, want the reloaded content", got)
	}
	if err := e.Save(); err != nil {
		t.Errorf("Save() after reload = %v, want nil", err)
	}
}

// The save is atomic, so a write that cannot complete leaves the previous
// draft intact rather than a truncated file.
func TestAFailedSaveLeavesTheOriginalAlone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nota.md")
	if err := os.WriteFile(path, []byte("el borrador bueno"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New()
	e.SetText("texto nuevo")
	e.Path = path
	e.Modified = true

	// Renaming onto a directory fails on every platform.
	blocked := filepath.Join(dir, "sub")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	e.Path = blocked

	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	// ForceSave, not Save: this is exercising the write failing, which has
	// nothing to do with ErrExternalChange — Save would also (correctly)
	// refuse here, but for the unrelated reason that blocked was never
	// loaded, muddying what this test is actually checking.
	if err := e.ForceSave(); err == nil {
		t.Fatal("saving onto a directory should fail")
	}
	if e.Modified != true {
		t.Error("a failed save cleared the modified flag, so the user would think it saved")
	}
	if b, _ := os.ReadFile(path); string(b) != "el borrador bueno" {
		t.Errorf("the original file changed: %q", b)
	}
	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Errorf("directory has %d entries after a failed save, want %d (the temp file was left behind)", len(after), len(before))
	}
}

func TestSaveAsKeepsTheOldNameWhenItFails(t *testing.T) {
	e := New()
	e.SetText("texto")
	e.Path = "original.md"

	err := e.SaveAs(filepath.Join(t.TempDir(), "no-existe", "x.md"))

	if err == nil {
		t.Fatal("saving into a missing directory should fail")
	}
	if e.Path != "original.md" {
		t.Errorf("Path = %q, want the previous name back", e.Path)
	}
}

// Naming a file that does not exist yet is how you start a document: the name
// has to stick, or ctrl+s would ask for it again.
func TestLoadingAMissingFileKeepsTheName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todavia-no.md")
	e := New()

	if err := e.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if e.Path != path {
		t.Errorf("Path = %q, want %q", e.Path, path)
	}
	if e.Text() != "" || e.Modified {
		t.Errorf("document is %q (modified=%v), want empty and clean", e.Text(), e.Modified)
	}
}

// Load keeps the document's bytes as LF internally, so wrapping, the cursor
// and the stats panel never see a \r sitting in the buffer as a stray glyph.
func TestLoadNormalizesCRLF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "windows.md")
	if err := os.WriteFile(path, []byte("uno\r\ndos\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New()
	if err := e.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := e.Text(); got != "uno\ndos\n" {
		t.Errorf("Text() = %q, want %q", got, "uno\ndos\n")
	}

	// The stats panel counted a \r as a character on every line before this
	// fix; with the buffer normalized there is nothing left to over-count.
	lf := New()
	lf.SetText("uno\ndos\n")
	if got, want := e.CharCount(), lf.CharCount(); got != want {
		t.Errorf("CharCount() = %d, want %d (same as the LF-only equivalent)", got, want)
	}
}

// A file that came in as CRLF must go back out as CRLF, unmodified, or a
// Windows collaborator sees every line flagged as changed by their VCS.
func TestSaveRoundTripsCRLF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "windows.md")
	original := []byte("uno\r\ndos\r\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	e := New()
	if err := e.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := e.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Errorf("file is %q, want it unchanged at %q", got, original)
	}
}

// The reverse must hold too: an LF file stays LF, never picking up a \r it
// never had.
func TestSaveRoundTripsLF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unix.md")
	original := []byte("uno\ndos\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	e := New()
	if err := e.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := e.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Errorf("file is %q, want it unchanged at %q", got, original)
	}
}

// A document that never went through Load — started fresh with ctrl+n — has
// no CRLF history to preserve, so it saves as LF.
func TestNewDocumentSavesWithLF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nuevo.md")
	e := New()
	e.SetText("uno\ndos\n")
	e.Path = path

	if err := e.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(got, []byte("\r")) {
		t.Errorf("file is %q, want no CR", got)
	}
}

// A file that mixes \r\n and bare \n has no single correct answer, so the
// policy is: any \r\n found at all means the whole document saves as CRLF.
func TestMixedLineEndingsPickCRLF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mixto.md")
	if err := os.WriteFile(path, []byte("uno\r\ndos\nzz\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New()
	if err := e.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := e.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := "uno\r\ndos\r\nzz\r\n"; string(got) != want {
		t.Errorf("file is %q, want %q", got, want)
	}
}

func TestLoadReportsRealErrors(t *testing.T) {
	e := New()

	if err := e.Load(t.TempDir()); err == nil {
		t.Error("opening a directory should be an error, not an empty document")
	}
}

func TestIsBinary(t *testing.T) {
	dir := t.TempDir()
	for _, c := range []struct {
		name    string
		content []byte
		want    bool
	}{
		{"plain text", []byte("hola"), false},
		{"utf-8 with accents is text", []byte("el niño escribió"), false},
		{"empty file", []byte{}, false},
		{"a NUL byte means binary", []byte("\x89PNG\x00\x1a"), true},
	} {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(dir, c.name)
			if err := os.WriteFile(path, c.content, 0o644); err != nil {
				t.Fatal(err)
			}

			if got := IsBinary(path); got != c.want {
				t.Errorf("IsBinary = %v, want %v", got, c.want)
			}
		})
	}
}
