//go:build unix

package ui

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A named pipe blocks open() until someone writes to it. The sniff must not
// touch it, and must still hide the PNG beside it.
func TestSniffSkipsPipesAndStillHidesBinaries(t *testing.T) {
	dir := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(dir, "pipe"), 0o644); err != nil {
		t.Skip("no mkfifo:", err)
	}
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("hola"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.png"), []byte("\x89PNG\x00\x00"), 0o644)

	var e explorer
	done := make(chan struct{})
	go func() { applySniff(&e, e.refresh(dir, "")); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("sniff blocked on the named pipe")
	}
	if got := strings.Join(names(&e), " "); strings.Contains(got, "b.png") || !strings.Contains(got, "a.md") {
		t.Errorf("listing = %q, want a.md without b.png", got)
	}
}
