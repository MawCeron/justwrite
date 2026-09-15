package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A directory with one of everything the listing has to deal with.
func sampleDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, content []byte) {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("alpha.md", []byte("uno"))
	write("beta.md", []byte("dos"))
	write("gamma.txt", []byte("tres"))
	write(".oculto", []byte("secreto"))
	write("imagen.png", []byte("\x89PNG\x00\x1a")) // a NUL byte: binary
	if err := os.Mkdir(filepath.Join(dir, "carpeta"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func names(e *explorer) []string {
	var out []string
	for _, it := range e.shown {
		out = append(out, it.name)
	}
	return out
}

// Opening a PNG fills the page with control characters, and dotfiles are not
// what anyone came here to write. Both stay out of the way until asked for.
func TestNoiseIsHiddenUntilAskedFor(t *testing.T) {
	var e explorer
	e.refresh(sampleDir(t), "")

	got := strings.Join(names(&e), " ")
	for _, unwanted := range []string{".oculto", "imagen.png"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("%s should not be listed by default: %s", unwanted, got)
		}
	}
	for _, wanted := range []string{"carpeta/", "alpha.md", "gamma.txt"} {
		if !strings.Contains(got, wanted) {
			t.Errorf("%s is missing from the listing: %s", wanted, got)
		}
	}

	e.toggleAll()

	got = strings.Join(names(&e), " ")
	if !strings.Contains(got, ".oculto") || !strings.Contains(got, "imagen.png") {
		t.Errorf("the toggle did not reveal everything: %s", got)
	}
}

func TestDirectoriesComeFirst(t *testing.T) {
	var e explorer
	e.refresh(sampleDir(t), "")

	list := names(&e)
	if list[0] != "../" || list[1] != "carpeta/" {
		t.Errorf("listing starts %v, want ../ then the directory", list[:min(2, len(list))])
	}
}

func TestFilteringNarrowsTheListAndCounts(t *testing.T) {
	var e explorer
	e.refresh(sampleDir(t), "")
	before := e.total

	e.setFilter("alp")

	if got := names(&e); len(got) != 1 || got[0] != "alpha.md" {
		t.Fatalf("filtered list = %v, want just alpha.md", got)
	}
	if want := "/alp  1/" + itoa(before); !strings.HasPrefix(e.footer(), want) {
		t.Errorf("footer = %q, want it to start %q", e.footer(), want)
	}

	e.setFilter("")

	if len(e.shown) != before {
		t.Errorf("clearing the filter left %d entries, want %d back", len(e.shown), before)
	}
}

// Going one directory too deep should cost one keypress, not your place in the
// listing: stepping up lands on the directory you just came out of.
func TestGoingUpLandsOnTheDirectoryYouLeft(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "carpeta")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}

	var e explorer
	e.refresh(parent, filepath.Base(child))

	it, ok := e.selected()
	if !ok || strings.TrimSuffix(it.name, "/") != "carpeta" {
		t.Errorf("cursor is on %q, want carpeta", it.name)
	}
}

func TestMoveStaysInsideTheList(t *testing.T) {
	var e explorer
	e.refresh(sampleDir(t), "")

	e.move(-5, 10)
	if e.cursor != 0 {
		t.Errorf("cursor = %d after moving up past the top, want 0", e.cursor)
	}

	e.move(100, 10)
	if want := len(e.shown) - 1; e.cursor != want {
		t.Errorf("cursor = %d after moving down past the end, want %d", e.cursor, want)
	}
}

// A file no longer on disk drops off the list instead of showing up as
// something that cannot actually be opened.
func TestRecentEntriesSkipsWhatNoLongerExists(t *testing.T) {
	dir := t.TempDir()
	gone := filepath.Join(dir, "gone.md")
	here := filepath.Join(dir, "here.md")
	if err := os.WriteFile(here, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// gone.md is recorded but never written to disk — as if it were deleted
	// after justwrite last had it open.

	now := time.Now()
	st := state{Documents: map[string]docState{
		gone: {Cursor: 0, Opened: now},
		here: {Cursor: 0, Opened: now.Add(-time.Minute)},
	}}

	entries := recentEntries(st)
	if len(entries) != 1 || entries[0].path != here {
		t.Fatalf("recentEntries = %+v, want just here.md", entries)
	}
}

// Newest first, and never more than maxRecentFiles.
func TestRecentEntriesOrderedAndCapped(t *testing.T) {
	dir := t.TempDir()
	st := state{Documents: map[string]docState{}}
	now := time.Now()
	for i := range maxRecentFiles + 5 {
		p := filepath.Join(dir, fmt.Sprintf("doc%02d.md", i))
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		// Higher i opened more recently.
		st.Documents[p] = docState{Cursor: i, Opened: now.Add(time.Duration(i) * time.Minute)}
	}

	entries := recentEntries(st)
	if len(entries) != maxRecentFiles {
		t.Fatalf("len(entries) = %d, want %d", len(entries), maxRecentFiles)
	}
	newest := filepath.Join(dir, fmt.Sprintf("doc%02d.md", maxRecentFiles+4))
	if entries[0].path != newest {
		t.Errorf("entries[0] = %q, want the most recently opened %q", entries[0].path, newest)
	}
	oldestKept := filepath.Join(dir, fmt.Sprintf("doc%02d.md", 5)) // the 5 oldest were dropped
	if entries[len(entries)-1].path != oldestKept {
		t.Errorf("entries[last] = %q, want %q", entries[len(entries)-1].path, oldestKept)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
