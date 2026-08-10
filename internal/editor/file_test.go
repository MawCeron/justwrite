package editor

import (
	"os"
	"path/filepath"
	"testing"
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
	// the writing directory with .tmp files on every ctrl+s.
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("the temporary file was left behind")
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

	if err := e.Save(); err == nil {
		t.Fatal("saving onto a directory should fail")
	}
	if e.Modified != true {
		t.Error("a failed save cleared the modified flag, so the user would think it saved")
	}
	if b, _ := os.ReadFile(path); string(b) != "el borrador bueno" {
		t.Errorf("the original file changed: %q", b)
	}
	if _, err := os.Stat(blocked + ".tmp"); !os.IsNotExist(err) {
		t.Error("the temporary file was left behind after a failure")
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
