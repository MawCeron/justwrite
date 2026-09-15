package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateRoundTrip(t *testing.T) {
	withTempConfigDir(t)

	if _, ok := loadState().cursorFor("/tmp/nowhere.md"); ok {
		t.Fatal("cursorFor found something before anything was recorded")
	}

	s := loadState()
	s.record("/tmp/nowhere.md", 42)
	if err := s.save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	got := loadState()
	cur, ok := got.cursorFor("/tmp/nowhere.md")
	if !ok || cur != 42 {
		t.Errorf("cursorFor = %d, %v, want 42, true", cur, ok)
	}
}

// Recording a relative path and looking it up by its absolute form (or the
// reverse) must land on the same entry — the same file opened two different
// ways is still one document.
func TestStateKeysByAbsolutePath(t *testing.T) {
	withTempConfigDir(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	s := state{Documents: map[string]docState{}}
	s.record("nota.md", 7)

	cur, ok := s.cursorFor(filepath.Join(dir, "nota.md"))
	if !ok || cur != 7 {
		t.Errorf("cursorFor(absolute) = %d, %v, want 7, true", cur, ok)
	}
}

// The map is capped, so a machine that has opened hundreds of files does not
// grow the state file without bound.
func TestStateEvictsTheOldestPastTheCap(t *testing.T) {
	withTempConfigDir(t)

	dir := t.TempDir()
	path := func(name string) string { return filepath.Join(dir, name) }

	s := state{Documents: map[string]docState{}}
	// Safely in the past, so record()'s own time.Now() below is unambiguously
	// later than every one of these on any clock, however coarse.
	base := time.Now().Add(-24 * time.Hour)
	for i := range maxStateEntries {
		abs, err := filepath.Abs(path(fmt.Sprintf("doc%02d.md", i)))
		if err != nil {
			t.Fatal(err)
		}
		s.Documents[abs] = docState{
			Cursor: i,
			Opened: base.Add(time.Duration(i) * time.Minute), // doc00 is the oldest
		}
	}

	s.record(path("new.md"), 999) // one more pushes past the cap

	if len(s.Documents) != maxStateEntries {
		t.Fatalf("len(Documents) = %d, want %d", len(s.Documents), maxStateEntries)
	}
	if _, ok := s.cursorFor(path("doc00.md")); ok {
		t.Error("the oldest entry survived eviction")
	}
	if cur, ok := s.cursorFor(path("new.md")); !ok || cur != 999 {
		t.Error("the newly recorded entry was evicted instead of the oldest")
	}
}
