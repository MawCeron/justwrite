package editor

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

// VisualLines breaks the document into rows of at most width columns, breaking
// at the last space that fits and hard-breaking a word that never fits.
//
// It always returns at least one line: an empty document still has a row for
// the cursor to sit on, and every caller indexes the result.
//
// ponytail: O(buffer) and recomputed per frame; cache per logical line if a
// long document ever feels slow.
func (e *Editor) VisualLines(width int) []VisualLine {
	if width < 1 {
		width = 1
	}
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

		for segStart := pos; ; {
			if logicalEnd-segStart <= width {
				lines = append(lines, VisualLine{segStart, logicalEnd})
				break
			}
			wrapAt := width
			for i := segStart + width - 1; i >= segStart; i-- {
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
