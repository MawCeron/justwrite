package editor

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ErrNoPath means the document has never been named, so the caller should ask
// for a filename instead of saving.
var ErrNoPath = errors.New("the document has no filename yet")

// ErrExternalChange means the file moved on disk since it was loaded — a
// sync, another process, another justwrite — so Save refused to write
// through it. The caller decides: reload and lose local edits, overwrite
// anyway with ForceSave, or save under a different name.
var ErrExternalChange = errors.New("the file on disk has changed since it was opened")

// Load reads path into the document. A file that does not exist yet is not an
// error: the name is kept so ctrl+s saves straight to it, instead of dropping
// the name and asking again for something the user already typed.
func (e *Editor) Load(path string) error {
	b, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		e.SetText("")
		e.cursor = 0
	case err != nil:
		return err
	default:
		text, crlf := normalizeLineEndings(string(b))
		e.SetText(text)
		e.crlf = crlf
	}
	e.Path = path
	e.Modified = false
	e.refreshLoadedStat()
	return nil
}

// refreshLoadedStat remembers the on-disk state Save compares against. The
// zero time stands for "did not exist" — a file that does not exist either
// side of that comparison is not a conflict, but one that exists on only one
// side is.
func (e *Editor) refreshLoadedStat() {
	info, err := os.Stat(e.Path)
	if err != nil {
		e.loadedModTime = time.Time{}
		return
	}
	e.loadedModTime = info.ModTime()
}

// externalChange reports whether the file has moved on disk since it was
// last loaded or saved.
func (e *Editor) externalChange() bool {
	info, err := os.Stat(e.Path)
	if err != nil {
		return !e.loadedModTime.IsZero()
	}
	return !info.ModTime().Equal(e.loadedModTime)
}

// normalizeLineEndings converts CRLF (and a stray lone CR) to LF for the
// in-memory buffer, and reports whether the file used CRLF at all. A file
// that mixes endings is treated as CRLF — once any \r\n is present, saving
// should not guess a third convention.
func normalizeLineEndings(s string) (text string, crlf bool) {
	crlf = strings.Contains(s, "\r\n")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s, crlf
}

// Save writes the document through a temporary file in the same directory
// and renames it into place, refusing if the file changed on disk since it
// was opened — see ErrExternalChange. The rename is atomic, so an
// interrupted or failed write can never leave a half-written draft where the
// finished one was.
func (e *Editor) Save() error {
	if e.Path == "" {
		return ErrNoPath
	}
	if e.externalChange() {
		return ErrExternalChange
	}
	return e.writeFile()
}

// ForceSave writes the document regardless of what changed on disk — the
// "overwrite anyway" choice after Save returns ErrExternalChange.
func (e *Editor) ForceSave() error {
	if e.Path == "" {
		return ErrNoPath
	}
	return e.writeFile()
}

func (e *Editor) writeFile() error {
	// Whatever is still mid-burst has to become part of the undo history
	// before saved is recorded against it, or undoing those same keystrokes
	// later would look like a return to this save rather than a departure
	// from it.
	e.CommitPending()

	mode := os.FileMode(0o644)
	if info, err := os.Stat(e.Path); err == nil {
		mode = info.Mode().Perm()
	}

	dir := filepath.Dir(e.Path)
	tmp, err := os.CreateTemp(dir, filepath.Base(e.Path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	text := e.Text()
	if e.crlf {
		text = strings.ReplaceAll(text, "\n", "\r\n")
	}
	if _, err := tmp.WriteString(text); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, e.Path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	syncDir(dir)
	e.saved = e.currentSeq()
	e.Modified = false
	e.refreshLoadedStat()
	return nil
}

// syncDir best-effort fsyncs dir so the rename above survives a power loss,
// not just a crashed process. Windows has no equivalent, so this is a no-op
// there.
func syncDir(dir string) {
	if runtime.GOOS == "windows" {
		return
	}
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	defer d.Close()
	d.Sync()
}

// SaveAs names the document and saves it. The name only sticks if the write
// succeeded, so a failed save leaves the document exactly as it was.
//
// This skips the ErrExternalChange check: naming a document is a deliberate
// choice already confirmed at the dialog (overwriting an existing file there
// asks first), not the silent ctrl+s Save guards against.
func (e *Editor) SaveAs(path string) error {
	previous := e.Path
	e.Path = path
	if err := e.writeFile(); err != nil {
		e.Path = previous
		return err
	}
	return nil
}

// IsBinary reports whether a file looks like something other than text, using
// git's heuristic: a NUL byte within the first 8 KB. Without this, opening a
// PNG from the file dialog fills the page with garbage.
func IsBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	var head [8000]byte
	n, err := io.ReadFull(f, head[:])
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return false
	}
	return bytes.IndexByte(head[:n], 0) >= 0
}
