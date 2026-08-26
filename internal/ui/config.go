package ui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// config is the handful of settings the user can change from inside the app
// — never a file they are expected to open and edit themselves.
type config struct {
	SessionGoal int // words for this sitting; 0 means none set
	DocGoal     int // total words the document is aiming for; 0 means none set
}

// userConfigDir is os.UserConfigDir, indirected so tests can point it at a
// throwaway directory instead of reading and overwriting whatever the
// machine running them actually has configured.
var userConfigDir = os.UserConfigDir

// configPath is where those settings live on disk.
func configPath() (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "justwrite", "config"), nil
}

// loadConfig reads the settings file. A missing file is not an error — it
// just means nothing has been changed from the defaults yet.
func loadConfig() config {
	var c config
	path, err := configPath()
	if err != nil {
		return c
	}
	f, err := os.Open(path)
	if err != nil {
		return c
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			continue
		}
		switch strings.TrimSpace(key) {
		case "session_goal":
			c.SessionGoal = n
		case "doc_goal":
			c.DocGoal = n
		}
	}
	return c
}

// save persists every setting at once, so it is still there the next time
// justwrite opens. Written whole rather than patching one line in place —
// two settings today, and reading them both back in on the next start is
// cheaper than a parser that remembers where every other line was. A goal
// left at 0 (never set, or cleared) is simply omitted.
func (c config) save() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	if c.SessionGoal > 0 {
		fmt.Fprintf(&b, "session_goal = %d\n", c.SessionGoal)
	}
	if c.DocGoal > 0 {
		fmt.Fprintf(&b, "doc_goal = %d\n", c.DocGoal)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
