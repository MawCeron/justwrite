package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// maxStateEntries caps how many documents state remembers, oldest opened
// dropped first — a file that only ever grows is one nobody notices growing
// forever.
const maxStateEntries = 50

// docState is what justwrite remembers about one document between runs.
type docState struct {
	Cursor int       `json:"cursor"`
	Opened time.Time `json:"opened"`
}

// state is what justwrite persists about the documents it has opened —
// distinct from config. config is settings the user changes on purpose;
// state is internal bookkeeping nobody edits by hand, kept in its own file
// so the two can evolve independently. Autosave content (#16) and a recent
// files list (#19) both belong here too, once they exist: Opened already
// gives them the recency ordering a list of "recent" anything needs.
type state struct {
	Documents map[string]docState `json:"documents"`
}

// statePath is where that file lives on disk.
func statePath() (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "justwrite", "state"), nil
}

// loadState reads the state file. A missing or unreadable one is not an
// error — it just means nothing has been recorded yet.
func loadState() state {
	empty := state{Documents: map[string]docState{}}

	path, err := statePath()
	if err != nil {
		return empty
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return empty
	}

	var s state
	if err := json.Unmarshal(b, &s); err != nil || s.Documents == nil {
		return empty
	}
	return s
}

// save writes the whole file, the same way config does.
func (s state) save() error {
	path, err := statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// record remembers cursor for path, keyed by its absolute form so the same
// file opened two different ways is still one entry. Evicts the least
// recently opened document once the map grows past maxStateEntries.
func (s *state) record(path string, cursor int) {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	s.Documents[path] = docState{Cursor: cursor, Opened: time.Now()}

	for len(s.Documents) > maxStateEntries {
		var oldestPath string
		var oldest time.Time
		for p, d := range s.Documents {
			if oldestPath == "" || d.Opened.Before(oldest) {
				oldestPath, oldest = p, d.Opened
			}
		}
		delete(s.Documents, oldestPath)
	}
}

// cursorFor is where path was left, if justwrite has seen it before.
func (s state) cursorFor(path string) (int, bool) {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	d, ok := s.Documents[path]
	return d.Cursor, ok
}
