package ui

import (
	"slices"
	"strings"

	"github.com/MawCeron/justwrite/internal/editor"
	"github.com/charmbracelet/x/ansi"
)

const (
	// The page never grows past this many columns however wide the terminal
	// is: long lines of prose are hard to track back to the next line.
	measure = 82

	topMargin = 3
	bottomGap = 1

	minWidth  = 30
	minHeight = 10
)

func (a App) textWidth() int  { return max(min(measure, a.w-8), 1) }
func (a App) textHeight() int { return max(a.h-topMargin-bottomGap-1, 1) }

func (a App) View() string {
	if a.w < minWidth || a.h < minHeight {
		return textStyle.Render("terminal too small")
	}

	rows := make([]string, 0, a.h)
	for range topMargin {
		rows = append(rows, "")
	}
	rows = append(rows, a.page()...)
	for range bottomGap {
		rows = append(rows, "")
	}
	rows = append(rows, a.statusBar())

	if panel := a.panel(); panel != "" {
		rows = overlay(rows, panel, a.w)
	}
	return strings.Join(rows, "\n")
}

// page renders the visible rows of the document, centred.
func (a App) page() []string {
	width, height := a.textWidth(), a.textHeight()
	vlines := a.ed.VisualLines(width)
	a.ed.AdjustScroll(vlines, height)

	cursorRow := a.ed.CursorVisualLine(vlines)
	pad := strings.Repeat(" ", max((a.w-width)/2, 0))

	rows := make([]string, 0, height)
	for i := a.ed.Scroll; i < a.ed.Scroll+height; i++ {
		if i < 0 || i >= len(vlines) {
			rows = append(rows, "")
			continue
		}
		rows = append(rows, pad+a.renderLine(vlines[i], i == cursorRow))
	}
	return rows
}

// renderLine draws one row, painting the selection and the block cursor into
// it. Runs of identical styling are rendered together, so a full page costs a
// handful of styled writes rather than one per character.
//
// The cursor is only drawn on the row the cursor actually belongs to: where a
// line wraps, one position is both the end of a row and the start of the next,
// and drawing it on both puts two cursors on screen.
func (a App) renderLine(vl editor.VisualLine, isCursorRow bool) string {
	buf := a.ed.Runes()
	selStart, selEnd, hasSel := a.ed.Selection()
	cursor := -1
	if isCursorRow {
		cursor = a.ed.Cursor()
	}

	const (
		plain = iota
		selected
		atCursor
	)
	styleFor := func(class int) func(...string) string {
		switch class {
		case atCursor:
			return cursorStyle.Render
		case selected:
			return selectStyle.Render
		default:
			return textStyle.Render
		}
	}

	var out strings.Builder
	var run strings.Builder
	runClass := plain

	flush := func() {
		if run.Len() > 0 {
			out.WriteString(styleFor(runClass)(run.String()))
			run.Reset()
		}
	}

	for i := vl.Start; i <= vl.End; i++ {
		r := ' '
		switch {
		case i < vl.End:
			r = buf[i]
		case i != cursor:
			// Past the end of the row, with no cursor to park here.
			flush()
			return out.String()
		}

		class := plain
		switch {
		case i == cursor:
			class = atCursor
		case hasSel && i >= selStart && i < selEnd:
			class = selected
		}

		if class != runClass {
			flush()
			runClass = class
		}
		run.WriteRune(r)
	}
	flush()
	return out.String()
}

// overlay composites a panel over the middle of the screen, leaving whatever
// is beside it visible — that is what makes it read as a panel floating over
// the page rather than a screen of its own.
func overlay(rows []string, panel string, width int) []string {
	lines := strings.Split(panel, "\n")
	panelWidth := 0
	for _, l := range lines {
		panelWidth = max(panelWidth, ansi.StringWidth(l))
	}

	x := max((width-panelWidth)/2, 0)
	y := max((len(rows)-len(lines))/2, 0)

	out := slices.Clone(rows)
	for i, l := range lines {
		row := y + i
		if row < 0 || row >= len(out) {
			continue
		}
		out[row] = spliceInto(out[row], l, x)
	}
	return out
}

// spliceInto puts fg into bg starting at column x, keeping bg on both sides.
func spliceInto(bg, fg string, x int) string {
	left := ansi.Truncate(bg, x, "")
	if gap := x - ansi.StringWidth(left); gap > 0 {
		left += strings.Repeat(" ", gap)
	}
	right := ansi.TruncateLeft(bg, x+ansi.StringWidth(fg), "")
	return left + fg + right
}
