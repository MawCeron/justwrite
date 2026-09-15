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

	s := state{Documents: map[string]docState{}}
	now := time.Now()
	for i := range maxStateEntries {
		s.Documents[fmt.Sprintf("/tmp/doc/%02d.md", i)] = docState{
			Cursor: i,
			Opened: now.Add(time.Duration(i) * time.Minute), // doc/00 is the oldest
		}
	}

	s.record("/tmp/new.md", 999) // one more pushes past the cap

	if len(s.Documents) != maxStateEntries {
		t.Fatalf("len(Documents) = %d, want %d", len(s.Documents), maxStateEntries)
	}
	if _, ok := s.cursorFor("/tmp/doc/00.md"); ok {
		t.Error("the oldest entry survived eviction")
	}
	if cur, ok := s.cursorFor("/tmp/new.md"); !ok || cur != 999 {
		t.Error("the newly recorded entry was evicted instead of the oldest")
	}
}
