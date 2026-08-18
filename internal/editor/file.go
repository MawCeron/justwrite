package editor

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrNoPath means the document has never been named, so the caller should ask
// for a filename instead of saving.
var ErrNoPath = errors.New("the document has no filename yet")

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
		e.SetText(string(b))
	}
	e.Path = path
	e.Modified = false
	return nil
}

// Save writes the document through a temporary file in the same directory and
// renames it into place. The rename is atomic, so an interrupted or failed
// write can never leave a half-written draft where the finished one was.
func (e *Editor) Save() error {
	if e.Path == "" {
		return ErrNoPath
	}
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
	if _, err := tmp.WriteString(e.Text()); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
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
	e.Modified = false
	return nil
}

// SaveAs names the document and saves it. The name only sticks if the write
// succeeded, so a failed save leaves the document exactly as it was.
func (e *Editor) SaveAs(path string) error {
	previous := e.Path
	e.Path = path
	if err := e.Save(); err != nil {
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
