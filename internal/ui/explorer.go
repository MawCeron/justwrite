package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MawCeron/justwrite/internal/editor"
	"github.com/charmbracelet/x/ansi"
)

// maxRecentFiles is how many entries the open dialog's recent-files list
// shows — "ten paths" per the issue this is built from.
const maxRecentFiles = 10

type entry struct {
	name      string // shown as-is, with a trailing / for directories
	path      string
	isDir     bool
	isUp      bool
	noise     bool // hidden or binary: out of the way unless asked for
	sniffable bool // might be a regular file: worth opening to look for binary content
	recent    bool // a path from elsewhere, not a name in the current directory
}

// explorer is the directory listing behind the open and save dialogs.
type explorer struct {
	dir     string
	all     []entry
	shown   []entry
	cursor  int
	offset  int
	filter  string
	showAll bool
	recent  bool // showing recentEntries instead of e.dir's contents
	total   int  // entries before the filter, for the counter
}

// refresh reads dir, without opening any file in it, and returns the command
// that sniffs the files for binary content to hide. focus is the name to put the cursor on, which is how
// stepping up to the parent lands on the directory you just came out of
// instead of dumping you at the top of the list.
func (e *explorer) refresh(dir, focus string) tea.Cmd {
	abs, err := filepath.Abs(dir)
	if err == nil {
		dir = abs
	}
	e.dir = dir
	e.all = nil
	e.filter = ""
	e.recent = false

	if parent := filepath.Dir(dir); parent != dir {
		e.all = append(e.all, entry{name: "../", path: parent, isDir: true, isUp: true})
	}

	items, _ := os.ReadDir(dir) // an unreadable directory simply lists as empty
	var dirs, files []entry
	for _, it := range items {
		p := filepath.Join(dir, it.Name())
		hidden := strings.HasPrefix(it.Name(), ".")
		if it.IsDir() {
			dirs = append(dirs, entry{name: it.Name() + "/", path: p, isDir: true, noise: hidden})
			continue
		}
		// Binary files are hidden once sniff reports back; until then they list.
		// A FIFO, socket or device would block the sniff's open(); a symlink
		// may point at one, so the sniff itself checks where it leads.
		t := it.Type()
		files = append(files, entry{name: it.Name(), path: p, noise: hidden,
			sniffable: t.IsRegular() || t&os.ModeSymlink != 0})
	}

	byName := func(s []entry) {
		sort.Slice(s, func(i, j int) bool {
			return strings.ToLower(s[i].name) < strings.ToLower(s[j].name)
		})
	}
	byName(dirs)
	byName(files)

	e.all = append(e.all, dirs...)
	e.all = append(e.all, files...)
	e.apply()
	e.focusOn(focus)
	return e.sniff()
}

// sniffed carries the result of sniffing one directory's files for binary
// content, back to the update loop.
type sniffed struct {
	dir    string
	binary map[string]bool // by path
}

// sniffBatch is how many files one sniff command reads: a file on a hung mount
// stalls its batch, not the whole directory.
const sniffBatch = 16

// sniff reads the listing's regular files off the UI thread, in concurrent
// batches, and reports each batch's binaries as it finishes.
func (e *explorer) sniff() tea.Cmd {
	dir := e.dir
	var cmds []tea.Cmd
	var paths []string
	flush := func() {
		if len(paths) == 0 {
			return
		}
		batch := paths
		paths = nil
		cmds = append(cmds, func() tea.Msg {
			binary := map[string]bool{}
			for _, p := range batch {
				if info, err := os.Stat(p); err == nil && info.Mode().IsRegular() && editor.IsBinary(p) {
					binary[p] = true
				}
			}
			return sniffed{dir: dir, binary: binary}
		})
	}
	for _, it := range e.all {
		if it.sniffable && !it.isDir && !it.noise {
			if paths = append(paths, it.path); len(paths) == sniffBatch {
				flush()
			}
		}
	}
	flush()
	return tea.Batch(cmds...)
}

// markBinary applies a sniff result to the listing, unless it has moved on
// since: another directory, or the recent list, which has no binaries to hide.
// The cursor stays on the entry it was on, however many rows vanish above it:
// on a slow disk the result lands seconds in, long after the user has moved.
func (e *explorer) markBinary(s sniffed) {
	if s.dir != e.dir || e.recent {
		return
	}
	for i := range e.all {
		if s.binary[e.all[i].path] {
			e.all[i].noise = true
		}
	}
	sel, ok := e.selected()
	e.apply()
	if !ok {
		return
	}
	for i, it := range e.shown {
		if it.path == sel.path {
			e.cursor = i
			e.offset = clamp(e.offset, max(i-1, 0), i)
			return
		}
	}
}

// refreshRecent swaps the listing for entries, most recently opened first.
// Unlike a directory there is no parent to step up to and no name to sort
// by — the order already means something.
func (e *explorer) refreshRecent(entries []entry) {
	e.recent = true
	e.all = entries
	e.filter = ""
	e.apply()
	e.cursor, e.offset = 0, 0
}

// recentEntries is the maxRecentFiles most recently opened documents that
// still exist, newest first. Existence is checked here, on read, rather
// than when state is written, so a moved or deleted file quietly drops off
// the list instead of needing a cleanup pass of its own.
func recentEntries(st state) []entry {
	type ranked struct {
		path   string
		opened time.Time
	}
	all := make([]ranked, 0, len(st.Documents))
	for path, d := range st.Documents {
		all = append(all, ranked{path, d.Opened})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].opened.After(all[j].opened) })

	entries := make([]entry, 0, maxRecentFiles)
	for _, r := range all {
		if len(entries) == maxRecentFiles {
			break
		}
		if _, err := os.Stat(r.path); err != nil {
			continue // moved or deleted since it was last open
		}
		entries = append(entries, entry{name: r.path, path: r.path, recent: true})
	}
	return entries
}

// apply rebuilds the visible list from the filter and the noise setting.
func (e *explorer) apply() {
	needle := strings.ToLower(e.filter)
	e.shown, e.total = nil, 0
	for _, it := range e.all {
		if it.noise && !e.showAll {
			continue
		}
		e.total++ // what the filter is narrowing down from
		if needle != "" && !strings.Contains(strings.ToLower(it.name), needle) {
			continue
		}
		e.shown = append(e.shown, it)
	}
	e.cursor = clamp(e.cursor, 0, max(len(e.shown)-1, 0))
	e.offset = clamp(e.offset, 0, max(len(e.shown)-1, 0))
}

func (e *explorer) focusOn(name string) {
	e.cursor, e.offset = 0, 0
	if name == "" {
		return
	}
	for i, it := range e.shown {
		if strings.TrimSuffix(it.name, "/") == name {
			e.cursor = i
			return
		}
	}
}

func (e *explorer) selected() (entry, bool) {
	if e.cursor < 0 || e.cursor >= len(e.shown) {
		return entry{}, false
	}
	return e.shown[e.cursor], true
}

func (e *explorer) move(delta, window int) {
	e.cursor = clamp(e.cursor+delta, 0, max(len(e.shown)-1, 0))
	e.offset = clamp(e.offset, max(e.cursor-window+1, 0), e.cursor)
	e.offset = clamp(e.offset, 0, max(len(e.shown)-window, 0))
}

func (e *explorer) setFilter(s string) {
	e.filter = s
	e.apply()
	e.cursor, e.offset = 0, 0
}

func (e *explorer) toggleAll() {
	e.showAll = !e.showAll
	e.apply()
}

// rows renders n rows of the listing.
func (e *explorer) rows(n, width int, focused bool) []string {
	if len(e.shown) == 0 {
		out := []string{overlayDimStyle.Render("  (empty)")}
		for len(out) < n {
			out = append(out, "")
		}
		return out
	}

	out := make([]string, 0, n)
	for i := e.offset; i < e.offset+n; i++ {
		if i >= len(e.shown) {
			out = append(out, "")
			continue
		}
		it := e.shown[i]

		name := it.name
		if it.recent {
			// A path, not a name — truncate from the left like the dialog's
			// own title does, so the filename at the end stays legible
			// instead of the directory prefix that says less.
			name = dirLabel(name, width-2)
		}

		style, pad := overlayStyle, overlayStyle.Render
		if it.isDir {
			style = overlayDirStyle
		}
		if focused && i == e.cursor {
			style, pad = overlaySelStyle, overlaySelStyle.Render
			if it.isDir {
				style = overlaySelDirStyle
			}
		}
		out = append(out, fitCell(style.Render("  "+name), width, pad))
	}
	return out
}

// filePanel draws the open and save-as dialogs. They share the listing; save-as
// adds the filename field underneath it.
func (a App) filePanel() string {
	width := a.panelWidth()

	listRows := a.listRows()
	title := "open"
	if a.mode == ModeSaveAs {
		title = "save as"
	}

	body := a.exp.rows(listRows, width, a.mode == ModeOpen || !a.onName)

	if a.mode == ModeSaveAs {
		label := overlayDimStyle.Render("  name  ")
		if a.onName {
			label = overlayStyle.Render("  name  ")
		}
		body = append(body, divider, fitCell(label+a.name.View(), width, overlayStyle.Render))
	}

	label := title + " · " + dirLabel(a.exp.dir, width-ansi.StringWidth(title)-9)
	if a.exp.recent {
		label = title + " · recent"
	}

	footer := a.exp.footer()
	if footer == "" && a.mode == ModeOpen {
		footer = "ctrl+r recent"
	}
	return panelBox(label, footer, body, width)
}

// dirLabel shortens the current path from the left. The tail is the part that
// says where you are; the drive letter never is.
func dirLabel(dir string, room int) string {
	room = max(room, 8)
	if ansi.StringWidth(dir) <= room {
		return dir
	}
	return "…" + ansi.TruncateLeft(dir, ansi.StringWidth(dir)-room+1, "")
}

func (e *explorer) footer() string {
	if e.filter != "" {
		return fmt.Sprintf("/%s  %d/%d", e.filter, len(e.shown), e.total)
	}
	if e.recent {
		return "ctrl+r back"
	}
	if e.showAll {
		return "all files"
	}
	return ""
}
