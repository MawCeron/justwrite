package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/MawCeron/justwrite/internal/editor"
	"github.com/charmbracelet/x/ansi"
)

type entry struct {
	name  string // shown as-is, with a trailing / for directories
	path  string
	isDir bool
	isUp  bool
	noise bool // hidden or binary: out of the way unless asked for
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
	total   int // entries before the filter, for the counter
}

// refresh reads dir. focus is the name to put the cursor on, which is how
// stepping up to the parent lands on the directory you just came out of
// instead of dumping you at the top of the list.
func (e *explorer) refresh(dir, focus string) {
	abs, err := filepath.Abs(dir)
	if err == nil {
		dir = abs
	}
	e.dir = dir
	e.all = nil
	e.filter = ""

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
		// ponytail: binary sniffing reads every file once per directory, on the
		// UI thread. Move it into a tea.Cmd if a network share ever stalls here.
		files = append(files, entry{name: it.Name(), path: p, noise: hidden || editor.IsBinary(p)})
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
		out = append(out, fitCell(style.Render("  "+it.name), width, pad))
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
	return panelBox(label, a.exp.footer(), body, width)
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
	if e.showAll {
		return "all files"
	}
	return ""
}
