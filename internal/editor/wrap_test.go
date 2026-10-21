package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func rows(e *Editor, width int) []string {
	var out []string
	for _, vl := range e.VisualLines(width) {
		out = append(out, string(e.buf[vl.Start:vl.End]))
	}
	return out
}

func TestVisualLines(t *testing.T) {
	for _, c := range []struct {
		name  string
		text  string
		width int
		want  []string
	}{
		{"an empty document still has a row for the cursor", "", 10, []string{""}},
		{"a short line is not touched", "hola", 10, []string{"hola"}},
		{"wrapping breaks at the last space that fits", "uno dos tres", 7, []string{"uno ", "dos ", "tres"}},
		{"a word longer than the page breaks hard", "abcdefghij", 4, []string{"abcd", "efgh", "ij"}},
		{"explicit newlines split lines", "a\nb", 10, []string{"a", "b"}},
		{"a trailing newline leaves an empty last row", "a\n", 10, []string{"a", ""}},
		{"a blank line between paragraphs survives", "a\n\nb", 10, []string{"a", "", "b"}},

		// Every caller indexes row zero, so a terminal narrow enough to leave
		// no room for text still has to come back with a row.
		{"width zero still yields one row", "hola", 0, []string{"h", "o", "l", "a"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := New()
			e.SetText(c.text)

			got := rows(e, c.width)

			if strings.Join(got, "|") != strings.Join(c.want, "|") {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestVisualLinesNeverReturnsNothing(t *testing.T) {
	for _, width := range []int{-1, 0, 1, 80} {
		e := New()
		if n := len(e.VisualLines(width)); n == 0 {
			t.Errorf("width %d produced no rows, which would panic every caller", width)
		}
	}
}

// Fifty numbered lines of equal length, so line k starts at k*7.
func fiftyLines() *Editor {
	var lines []string
	for i := range 50 {
		lines = append(lines, fmt.Sprintf("line%02d", i))
	}
	e := New()
	e.SetText(strings.Join(lines, "\n"))
	return e
}

func TestAdjustScroll(t *testing.T) {
	const height = 10

	t.Run("the cursor near the top does not scroll", func(t *testing.T) {
		e := fiftyLines()
		e.cursor = 0

		e.AdjustScroll(e.VisualLines(20), height)

		if e.Scroll != 0 {
			t.Errorf("Scroll = %d, want 0", e.Scroll)
		}
	})

	t.Run("moving down keeps a margin below the cursor", func(t *testing.T) {
		e := fiftyLines()
		e.cursor = 30 * 7

		e.AdjustScroll(e.VisualLines(20), height)

		// Line 30 visible with 3 rows of context under it: 30+3+1-10.
		if want := 24; e.Scroll != want {
			t.Errorf("Scroll = %d, want %d", e.Scroll, want)
		}
	})

	t.Run("moving back up keeps a margin above the cursor", func(t *testing.T) {
		e := fiftyLines()
		e.Scroll = 24
		e.cursor = 20 * 7

		e.AdjustScroll(e.VisualLines(20), height)

		if want := 17; e.Scroll != want {
			t.Errorf("Scroll = %d, want %d", e.Scroll, want)
		}
	})

	t.Run("the end of the document does not scroll past the last row", func(t *testing.T) {
		e := fiftyLines()
		e.cursor = e.Len()

		e.AdjustScroll(e.VisualLines(20), height)

		if want := 40; e.Scroll != want { // 50 rows, 10 visible
			t.Errorf("Scroll = %d, want %d", e.Scroll, want)
		}
	})
}

// Rows are measured in terminal cells, not runes: a CJK character takes two,
// a combining mark none. Counting runes let a row grow past the page and the
// terminal re-wrapped it, throwing the cursor and the centring off.
func TestVisualLinesMeasuresCells(t *testing.T) {
	for _, c := range []struct {
		name  string
		text  string
		width int
		want  []string
	}{
		{"CJK takes two cells a rune", "你好世界你好", 6, []string{"你好世", "界你好"}},
		{"an emoji takes two cells", "😀😀😀😀", 4, []string{"😀😀", "😀😀"}},
		{"a combining mark takes none", "ééé", 2, []string{"éé", "é"}},
		{"wrapping still prefers the last space", "你好 世界你好", 6, []string{"你好 ", "世界你", "好"}},
		{"a rune wider than the page still takes a row", "你好", 1, []string{"你", "好"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := New()
			e.SetText(c.text)

			got := rows(e, c.width)

			if strings.Join(got, "|") != strings.Join(c.want, "|") {
				t.Errorf("got %q, want %q", got, c.want)
			}
			for _, row := range got {
				if w := ansi.StringWidth(row); w > max(c.width, 2) {
					t.Errorf("row %q is %d cells wide, page is %d", row, w, c.width)
				}
			}
		})
	}
}

// A tab is one rune but runs to the next tab stop. Counting it as one column
// let a row hold far more than the page, so tabs wrap by the cells they fill.
func TestVisualLinesExpandsTabs(t *testing.T) {
	for _, c := range []struct {
		name  string
		text  string
		width int
		want  []string
	}{
		{"a tab runs to the next stop", "a\tb\tc", 5, []string{"a\tb", "\tc"}},
		{"a tab at a stop takes a whole stop", "\t\t\t", 8, []string{"\t\t", "\t"}},
		{"a tab fits when the row has room", "a\tb", 10, []string{"a\tb"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := New()
			e.SetText(c.text)

			got := rows(e, c.width)

			if strings.Join(got, "|") != strings.Join(c.want, "|") {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
