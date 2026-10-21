package editor

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
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

// A grapheme cluster is one thing on screen however many runes it takes: a
// ZWJ family is five runes and two cells, a flag two runes and two cells.
// Measuring per rune summed them (six and four), wrapped far too early, and
// could break a row in the middle of one.
func TestVisualLinesMeasuresGraphemeClusters(t *testing.T) {
	const family = "👨‍👩‍👧"
	const flag = "🇲🇽"

	for _, c := range []struct {
		name  string
		text  string
		width int
		want  []string
	}{
		{"a ZWJ sequence is two cells", family + family + family, 4, []string{family + family, family}},
		{"a flag is two cells", flag + flag + flag, 4, []string{flag + flag, flag}},
		{"a mark after a space stays with it", "ab \u0301cd", 4, []string{"ab \u0301c", "d"}},
		{"a tab after a cluster still runs to the stop", flag + "\tb", 8, []string{flag + "\tb"}},
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

func TestRowWidths(t *testing.T) {
	got := RowWidths([]rune("a\tb👨‍👩‍👧é\t"))
	want := []int{1, 3, 1, 2, 0, 0, 0, 0, 1, 0, 4}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("RowWidths = %v, want %v", got, want)
	}
}

// A novel's worth of Spanish prose in paragraphs of a few hundred runes, with
// one line of CJK among them, wrapped on every keystroke.
func BenchmarkVisualLines(b *testing.B) {
	para := strings.Repeat("Había una vez un párrafo que se repite hasta el cansancio, con acentos y ñ. ", 5)
	e := New()
	e.SetText(strings.Repeat(para+"\n", 800) + "你好世界你好世界\n" + strings.Repeat(para+"\n", 800))
	b.ResetTimer()
	for range b.N {
		e.wrapped = nil // every call is a change, not a cache hit
		e.VisualLines(72)
	}
}

// The cache is keyed on the text itself, so any edit has to show up in the
// next answer — including one that leaves the length and the width alone.
func TestVisualLinesCacheSeesEveryEdit(t *testing.T) {
	e := New()
	e.SetText("uno dos tres")
	if got := rows(e, 7); strings.Join(got, "|") != "uno |dos |tres" {
		t.Fatalf("before the edit: %q", got)
	}

	e.SetCursor(3)
	e.DeleteForward() // "unodos tres": same width, different rows
	if got := rows(e, 7); strings.Join(got, "|") != "unodos |tres" {
		t.Errorf("after deleting: %q", got)
	}

	e.SetText("unodos tres") // equal text, new document: still right
	if got := rows(e, 7); strings.Join(got, "|") != "unodos |tres" {
		t.Errorf("after reloading: %q", got)
	}
	if got := rows(e, 20); strings.Join(got, "|") != "unodos tres" {
		t.Errorf("at another width: %q", got)
	}
}

// Whatever the text, a row measured the way the page draws it, by RowWidths,
// fits the width the wrapper was given — unless it is a single cluster wider
// than the page, which has nowhere to go. The wrapper measures a whole line and
// the renderer a row at a time, so a break that lands inside a cluster, or
// leaves a mark orphaned at the start of a row, makes them disagree.
func TestRowsFitTheirWidthWhateverTheText(t *testing.T) {
	alphabet := []rune{'a', 'b', ' ', ' ', '́', '你', '😀', '\t', '‍'}
	rng := rand.New(rand.NewPCG(1, 2))

	for range 3000 {
		text := make([]rune, 1+rng.IntN(30))
		for i := range text {
			text[i] = alphabet[rng.IntN(len(alphabet))]
		}
		e := New()
		e.SetText(string(text))

		for width := 3; width <= 8; width++ {
			for _, vl := range e.VisualLines(width) {
				row := e.buf[vl.Start:vl.End]
				total := 0
				for _, w := range RowWidths(row) {
					total += w
				}
				if vl.Start > 0 && e.buf[vl.Start-1] != '\n' && len(row) > 0 && joinsPrevious(e.buf[vl.Start-1], row[0]) {
					t.Fatalf("text %q at width %d: row %q starts mid-cluster", string(text), width, string(row))
				}
				if total > width && len(row) > 1 && !onlyOneCluster(row) {
					t.Fatalf("text %q at width %d: row %q is %d cells wide", string(text), width, string(row), total)
				}
			}
		}
	}
}

func onlyOneCluster(row []rune) bool {
	w := clusterWidths(row)
	return w[0] > 0 && !slices.ContainsFunc(w[1:], func(n int) bool { return n > 0 })
}

// joinsPrevious reports whether next is part of the grapheme cluster prev
// starts, so a row break between them cuts a cluster in two.
func joinsPrevious(prev, next rune) bool {
	cluster, _, _, _ := uniseg.FirstGraphemeClusterInString(string([]rune{prev, next}), -1)
	return utf8.RuneCountInString(cluster) == 2
}
