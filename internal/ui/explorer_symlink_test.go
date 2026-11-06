package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDialogEntersSymlinkedDirectory(t *testing.T) {
	for _, naming := range []bool{false, true} {
		for _, relative := range []bool{false, true} {
			dir := t.TempDir()
			target := filepath.Join(dir, "target")
			if err := os.Mkdir(target, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target, "chapter.md"), []byte("text"), 0o644); err != nil {
				t.Fatal(err)
			}
			if relative {
				target = "target"
			}
			link := filepath.Join(dir, "linked")
			if err := os.Symlink(target, link); err != nil {
				t.Skip("symlinks unavailable:", err)
			}
			a := testApp(t)
			a.exp.refresh(dir, "linked")
			selected, ok := a.exp.selected()
			if !ok || !selected.isDir || selected.name != "linked/" || selected.sniffable {
				t.Fatalf("linked directory classified as %+v", selected)
			}
			a, _ = a.enterSelection(naming)
			if a.exp.dir != link {
				t.Fatalf("dialog directory = %q, want %q", a.exp.dir, link)
			}
			a.exp.focusOn("chapter.md")
			selected, ok = a.exp.selected()
			if !ok || selected.path != filepath.Join(link, "chapter.md") {
				t.Fatalf("linked directory contents unavailable: %+v", selected)
			}
		}
	}
}

func TestNonDirectorySymlinksRemainFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "target.md"), []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{"file.md": "target.md", "broken": "missing", "cycle": "cycle"} {
		if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
			t.Skip("symlinks unavailable:", err)
		}
	}
	var e explorer
	applySniff(&e, e.refresh(dir, ""))
	for _, name := range []string{"file.md", "broken", "cycle"} {
		e.focusOn(name)
		selected, ok := e.selected()
		if !ok || selected.name != name || selected.isDir {
			t.Fatalf("non-directory symlink classified as %+v", selected)
		}
	}
}
