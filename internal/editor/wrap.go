package editor

import (
	"slices"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

// VisualLine is one rendered row: the slice of the document that lands on it.
// Wrapping is soft, so a logical line becomes several VisualLines without any
// character being added to the document.
type VisualLine struct {
	Start int
	End   int
}

// scrollMargin keeps this many rows of context between the cursor and the edge
// of the viewport, so typing at the bottom does not sit on the last row.
const scrollMargin = 3

// TabStop is the width of a tab stop, and of what the Tab key inserts.
const TabStop = 4

// tabWidth is how far a tab at column col runs, to the next tab stop.
func tabWidth(col int) int { return TabStop - col%TabStop }

// clusterWidths is the cells each rune of rs takes, measured by grapheme
// cluster: the cluster's whole width lands on its first rune and the rest take
// none, so a ZWJ emoji or a flag is two cells rather than the sum of its parts,
// and a row can never break inside one. A tab reports zero; it needs the
// column it lands on, which only the row knows.
func clusterWidths(rs []rune) []int {
	widths := make([]int, len(rs))
	for s, i := string(rs), 0; s != ""; {
		var cluster string
		cluster, s, _, _ = uniseg.FirstGraphemeClusterInString(s, -1)
		widths[i] = ansi.StringWidth(cluster)
		i += utf8.RuneCountInString(cluster)
	}
	return widths
}

// RowWidths is the cells each rune of one rendered row takes. The document
// keeps its tab bytes; only the page expands them, each to the next tab stop
// counted from the start of the row.
func RowWidths(row []rune) []int {
	widths := clusterWidths(row)
	for col, i := 0, 0; i < len(row); i++ {
		if row[i] == '\t' {
			widths[i] = tabWidth(col)
		}
		col += widths[i]
	}
	return widths
}

// VisualLines breaks the document into rows of at most width cells, breaking
// at the last space that fits and hard-breaking a word that never fits.
//
// It always returns at least one line: an empty document still has a row for
// the cursor to sit on, and every caller indexes the result.
//
// The answer is cached against the exact text and width it was computed for,
// so the several calls a keystroke makes — scrolling, the key handler, the
// render — wrap the document once, and moving the cursor wraps it not at all.
// The slice is shared: callers must not modify it.
func (e *Editor) VisualLines(width int) []VisualLine {
	if width < 1 {
		width = 1
	}
	if e.wrapped != nil && e.wrappedWidth == width && slices.Equal(e.wrappedBuf, e.buf) {
		return e.wrapped
	}
	e.wrapped = e.wrap(width)
	e.wrappedWidth = width
	e.wrappedBuf = append(e.wrappedBuf[:0], e.buf...)
	return e.wrapped
}

func (e *Editor) wrap(width int) []VisualLine {
	var lines []VisualLine
	n := len(e.buf)

	for pos := 0; pos <= n; {
		if pos == n {
			lines = append(lines, VisualLine{pos, pos})
			break
		}

		// The logical line runs to the next newline, or to the end.
		logicalEnd := n
		hasNewline := false
		for i := pos; i < n; i++ {
			if e.buf[i] == '\n' {
				logicalEnd, hasNewline = i, true
				break
			}
		}

		base := clusterWidths(e.buf[pos:logicalEnd])
		for segStart := pos; ; {
			// over is the first rune that no longer fits on this row.
			over, cells := segStart, 0
			for over < logicalEnd {
				if e.buf[over] == '\t' {
					cells += tabWidth(cells)
				} else {
					cells += base[over-pos]
				}
				if cells > width {
					break
				}
				over++
			}
			if over == logicalEnd {
				lines = append(lines, VisualLine{segStart, logicalEnd})
				break
			}
			wrapAt := max(over-segStart, 1) // a rune wider than the page still takes a row
			for i := over - 1; i >= segStart; i-- {
				if e.buf[i] == ' ' {
					wrapAt = i - segStart + 1 // the space stays on this row
					break
				}
			}
			lines = append(lines, VisualLine{segStart, segStart + wrapAt})
			segStart += wrapAt
		}

		if !hasNewline {
			break
		}
		pos = logicalEnd + 1
	}

	if len(lines) == 0 {
		lines = append(lines, VisualLine{0, 0})
	}
	return lines
}

// CursorVisualLine reports which row the cursor is on. The cursor may sit at
// the very end of a row, which is why the range check is inclusive.
func (e *Editor) CursorVisualLine(vlines []VisualLine) int {
	for i, vl := range vlines {
		if e.cursor >= vl.Start && e.cursor <= vl.End {
			return i
		}
	}
	return max(len(vlines)-1, 0)
}

// AdjustScroll moves the viewport the least it can to keep the cursor visible.
func (e *Editor) AdjustScroll(vlines []VisualLine, height int) {
	if height <= 0 || len(vlines) == 0 {
		return
	}
	cur := e.CursorVisualLine(vlines)

	if cur < e.Scroll+scrollMargin && e.Scroll > 0 {
		e.Scroll = max(cur-scrollMargin, 0)
	}
	if cur+scrollMargin >= e.Scroll+height {
		e.Scroll = min(max(cur+scrollMargin+1-height, 0), len(vlines)-1)
	}
	// Never leave blank rows below the last line when the document fills the
	// viewport.
	if e.Scroll+height > len(vlines) && len(vlines) >= height {
		e.Scroll = len(vlines) - height
	}
	e.Scroll = max(e.Scroll, 0)
}
