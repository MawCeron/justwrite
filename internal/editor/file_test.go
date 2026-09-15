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

func TestWriteSwapCreatesADotfileBesideTheDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nota.md")
	e := New()
	e.Path = path
	e.SetText("borrador")

	if err := e.WriteSwap(); err != nil {
		t.Fatalf("WriteSwap: %v", err)
	}

	swap := filepath.Join(filepath.Dir(path), ".nota.md.swp")
	b, err := os.ReadFile(swap)
	if err != nil {
		t.Fatalf("swap file: %v", err)
	}
	if string(b) != "borrador" {
		t.Errorf("swap content = %q, want %q", b, "borrador")
	}

	// The write goes through a temp file, same as a real save; nothing else
	// should be left behind in the directory.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("directory has %d entries after WriteSwap, want 1", len(entries))
	}
}

func TestWriteSwapIsANoOpWithoutAPath(t *testing.T) {
	e := New()
	e.SetText("sin nombre todavia")
	if err := e.WriteSwap(); err != nil {
		t.Errorf("WriteSwap on an unnamed document: %v", err)
	}
}

func TestRemoveSwapDeletesIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nota.md")
	e := New()
	e.Path = path
	e.SetText("borrador")
	if err := e.WriteSwap(); err != nil {
		t.Fatal(err)
	}

	if err := e.RemoveSwap(); err != nil {
		t.Fatalf("RemoveSwap: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), ".nota.md.swp")); !errors.Is(err, os.ErrNotExist) {
		t.Error("the swap file is still there")
	}

	// Neither a missing swap nor an unnamed document is an error.
	if err := e.RemoveSwap(); err != nil {
		t.Errorf("RemoveSwap with nothing to remove: %v", err)
	}
	if err := New().RemoveSwap(); err != nil {
		t.Errorf("RemoveSwap on an unnamed document: %v", err)
	}
}

// PendingSwap is what Load's caller checks afterward to decide whether to
// offer recovery: a swap newer than the document is worth having, one that
// is not is stale leftovers from before the last real save.
func TestPendingSwap(t *testing.T) {
	t.Run("newer swap is offered", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nota.md")
		if err := os.WriteFile(path, []byte("guardado"), 0o644); err != nil {
			t.Fatal(err)
		}
		touchLater(t, path)

		e := New()
		if err := e.Load(path); err != nil {
			t.Fatal(err)
		}
		e.SetText("autosalvado, más nuevo")
		if err := e.WriteSwap(); err != nil {
			t.Fatal(err)
		}
		touchLater(t, filepath.Join(filepath.Dir(path), ".nota.md.swp"))

		content, ok := e.PendingSwap()
		if !ok || content != "autosalvado, más nuevo" {
			t.Errorf("PendingSwap = %q, %v, want the swap content and true", content, ok)
		}
	})

	t.Run("a swap no newer than the document is stale and removed", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nota.md")
		if err := os.WriteFile(path, []byte("guardado"), 0o644); err != nil {
			t.Fatal(err)
		}
		e := New()
		e.Path = path
		e.SetText("vieja")
		if err := e.WriteSwap(); err != nil {
			t.Fatal(err)
		}
		touchLater(t, path) // the document is now newer than its own stale swap

		if _, ok := e.PendingSwap(); ok {
			t.Error("a stale swap was offered as pending")
		}
		if _, err := os.Stat(e.swapPath()); !errors.Is(err, os.ErrNotExist) {
			t.Error("the stale swap was not removed")
		}
	})

	t.Run("a swap for a document that no longer exists is still pending", func(t *testing.T) {
		e := New()
		e.Path = filepath.Join(t.TempDir(), "borrado.md")
		e.SetText("todo lo que queda")
		if err := e.WriteSwap(); err != nil {
			t.Fatal(err)
		}

		content, ok := e.PendingSwap()
		if !ok || content != "todo lo que queda" {
			t.Errorf("PendingSwap = %q, %v, want the swap content and true", content, ok)
		}
	})

	t.Run("no swap at all", func(t *testing.T) {
		e := New()
		e.Path = filepath.Join(t.TempDir(), "nota.md")
		if _, ok := e.PendingSwap(); ok {
			t.Error("PendingSwap found something out of nothing")
		}
	})
}

// touchLater sets path's mtime one second in the future, so a following
// comparison is unambiguous on filesystems with coarse mtime resolution.
func touchLater(t *testing.T, path string) {
	t.Helper()
	future := time.Now().Add(time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverSwapReplacesTheBufferAsModified(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nota.md")
	if err := os.WriteFile(path, []byte("en disco"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New()
	if err := e.Load(path); err != nil {
		t.Fatal(err)
	}

	e.RecoverSwap("recuperado")

	if e.Text() != "recuperado" {
		t.Errorf("Text() = %q, want %q", e.Text(), "recuperado")
	}
	if !e.Modified {
		t.Error("a recovered buffer must be marked modified")
	}
	// SetText clears loadedModTime; a save right after recovering must not
	// mistake the untouched file on disk for an external change.
	if err := e.Save(); err != nil {
		t.Errorf("Save right after recovering: %v", err)
	}
}
