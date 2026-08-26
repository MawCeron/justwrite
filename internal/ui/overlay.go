package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// panel returns the overlay for the current mode, or "" while writing.
func (a App) panel() string {
	switch a.mode {
	case ModeHelp:
		return a.helpPanel()
	case ModeAbout:
		return a.aboutPanel()
	case ModeStats:
		return a.statsPanel()
	case ModeConfirm:
		return a.confirmPanel()
	case ModeOpen, ModeSaveAs:
		return a.filePanel()
	}
	return ""
}

// panelWidth sizes overlays in cells rather than as a percentage of the
// terminal: a percentage gives a silly 100-column dialog on a wide monitor and
// a cramped one on a laptop.
// The last term is the floor guard: on a narrow terminal the panel gives up
// its margins rather than growing wider than the screen it sits on.
func (a App) panelWidth() int  { return min(max(a.w-8, 20), 64, a.w-2) }
func (a App) panelHeight() int { return clamp(a.h-6, 6, 20) }

// listRows is how many rows of the listing filePanel actually draws, so
// browse scrolls by the same window instead of recomputing a different one —
// save-as reserves two rows underneath for the filename field.
func (a App) listRows() int {
	if a.mode == ModeSaveAs {
		return max(a.panelHeight()-2, 1)
	}
	return a.panelHeight()
}

func clamp(v, lo, hi int) int { return min(max(v, lo), hi) }

// divider stands in for a body line that should be drawn as a rule joining
// both sides of the frame, rather than as content.
const divider = "\x00divider"

// panelBox frames body in a rounded border, with an optional label set into
// the top border and another into the bottom one.
func panelBox(title, footer string, body []string, width int) string {
	var out strings.Builder

	out.WriteString(overlayBorderStyle.Render("╭"))
	out.WriteString(borderRun(title, width, true))
	out.WriteString(overlayBorderStyle.Render("╮"))

	for _, line := range body {
		if line == divider {
			out.WriteString("\n" + overlayBorderStyle.Render("├"+strings.Repeat("─", width)+"┤"))
			continue
		}
		out.WriteString("\n" + overlayBorderStyle.Render("│"))
		out.WriteString(fitCell(line, width, overlayStyle.Render))
		out.WriteString(overlayBorderStyle.Render("│"))
	}

	out.WriteString("\n" + overlayBorderStyle.Render("╰"))
	out.WriteString(borderRun(footer, width, false))
	out.WriteString(overlayBorderStyle.Render("╯"))
	return out.String()
}

// borderRun draws one horizontal border, with the label tucked in at the left
// for a title and at the right for a footer.
func borderRun(label string, width int, atStart bool) string {
	dash := func(n int) string { return overlayBorderStyle.Render(strings.Repeat("─", max(n, 0))) }
	if label == "" || width < ansi.StringWidth(label)+6 {
		return dash(width)
	}
	// The title names the panel and gets the brighter ink; the footer is a
	// hint and stays quiet.
	tag := overlayDimStyle.Render(" " + label + " ")
	if atStart {
		tag = overlayTitleStyle.Render(" " + label + " ")
	}
	rest := width - ansi.StringWidth(tag) - 1
	if atStart {
		return dash(1) + tag + dash(rest)
	}
	return dash(rest) + tag + dash(1)
}

// fitCell pads or truncates s to exactly width columns, so a panel's edges
// stay straight no matter what is inside it.
func fitCell(s string, width int, pad func(...string) string) string {
	s = ansi.Truncate(s, width, "…")
	if gap := width - ansi.StringWidth(s); gap > 0 {
		s += pad(strings.Repeat(" ", gap))
	}
	return s
}

// ─── Help ────────────────────────────────────────────────────────────────────

// The shortcuts, in the order they are worth learning. {key, description}.
var shortcuts = [][2]string{
	{"ctrl+s", "save"},
	{"ctrl+o", "open a file"},
	{"ctrl+n", "new document"},
	{"ctrl+q", "quit"},
	{"ctrl+z", "undo"},
	{"ctrl+y", "redo"},
	{"ctrl+a", "select all"},
	{"ctrl+c", "copy"},
	{"ctrl+x", "cut"},
	{"ctrl+v", "paste"},
	{"ctrl+t", "word count & stats"},
	{"ctrl+←→", "jump a word"},
	{"ctrl+w", "delete a word"},
	{"shift+↑↓←→", "extend selection"},
	{"pgup/pgdn", "scroll a page"},
	{"shift+pgup/pgdn", "extend selection"},
	{"ctrl+home/end", "jump to start or end"},
	{"ctrl+shift+home/end", "extend to start or end"},
	{"tab", "insert 4 spaces"},
	{"esc", "close panel"},
}

// helpBandCount is how many two-column bands the shortcut list draws into,
// one card from the left half paired with one from the right.
func helpBandCount() int { return (len(shortcuts) + 1) / 2 }

// helpVisibleBands is how many of those bands fit in the panel at once — the
// two blank padding lines come out of the same budget as the cards, and each
// card is two lines tall (description, then key).
func (a App) helpVisibleBands() int {
	return max((a.panelHeight()-2)/2, 1)
}

// helpMaxScroll is the last scroll offset that still has a full window of
// bands to show, so scrolling can never run past the end of the list.
func helpMaxScroll(visible int) int {
	return max(helpBandCount()-visible, 0)
}

func (a App) helpPanel() string {
	width := a.panelWidth()

	// helpGap is the breathing room between the two card columns.
	const helpGap = 5

	// Each shortcut is a two-line card: the description on top, in the
	// brighter ink since that is what you scan for, and the key underneath
	// it in the quieter tone — a reference once you already know what you
	// want.
	colWidth := func(list [][2]string) int {
		w := 0
		for _, s := range list {
			w = max(w, ansi.StringWidth(s[0]), ansi.StringWidth(s[1]))
		}
		return w
	}
	desc := func(s [2]string, w int) string {
		return fitCell(overlayStyle.Render("  "+s[1]), w+2, overlayStyle.Render)
	}
	key := func(s [2]string, w int) string {
		return fitCell(overlayDimStyle.Render("  "+s[0]), w+2, overlayDimStyle.Render)
	}

	bands := helpBandCount()
	left, right := shortcuts[:bands], shortcuts[bands:]
	leftWidth, rightWidth := colWidth(left), colWidth(right)

	visible := a.helpVisibleBands()
	scroll := min(a.helpScroll, helpMaxScroll(visible))
	end := min(scroll+visible, bands)

	var body []string
	body = append(body, "")

	// Two columns only when both actually fit at their natural width — the
	// short "ctrl+X" column and the longer movement-key column need
	// different amounts of room, not an even half each.
	twoCol := width >= 2*2+leftWidth+rightWidth+helpGap
	kw := max(leftWidth, rightWidth)
	for i := scroll; i < end; i++ {
		if twoCol {
			descLine, keyLine := desc(left[i], leftWidth), key(left[i], leftWidth)
			if i < len(right) {
				gap := overlayStyle.Render(strings.Repeat(" ", helpGap))
				descLine += gap + desc(right[i], rightWidth)
				keyLine += gap + key(right[i], rightWidth)
			}
			body = append(body, descLine, keyLine)
		} else {
			body = append(body, desc(left[i], kw), key(left[i], kw))
			if i < len(right) {
				body = append(body, desc(right[i], kw), key(right[i], kw))
			}
		}
	}

	body = append(body, "")

	footer := "? about"
	if scroll > 0 || end < bands {
		footer = "↑↓ more  " + footer
	}
	return panelBox("help", footer, body, width)
}

// ─── About ───────────────────────────────────────────────────────────────────

func (a App) aboutPanel() string {
	width := a.panelWidth()
	title := VersionString()

	body := []string{
		"",
		overlayStyle.Render("  " + title),
		overlayDimStyle.Render("  A minimal, distraction-free text editor."),
		"",
		overlayDimStyle.Render("  github.com/MawCeron/justwrite"),
		// Kept in step with the LICENSE file, which is the one that counts.
		overlayDimStyle.Render("  © 2026 Mauricio Cerón · MIT"),
		"",
	}
	return panelBox("about", "", body, width)
}

// ─── Stats ───────────────────────────────────────────────────────────────────

func (a App) statsPanel() string {
	width := a.panelWidth()
	mins, underOne := a.ed.ReadingTime()
	read := fmt.Sprintf("~%d min", mins)
	if underOne {
		read = "< 1 min"
	}

	row := func(label, value string) string {
		return overlayDimStyle.Render(fmt.Sprintf("  %-12s", label)) + overlayStyle.Render(value)
	}
	body := []string{
		"",
		row("words", fmt.Sprint(a.ed.WordCount())),
		row("characters", fmt.Sprint(a.ed.CharCount())),
		row("pages", fmt.Sprint(a.ed.PageCount(a.textHeight(), a.textWidth()))),
		row("read time", read),
		"",
	}
	return panelBox("stats", "", body, width)
}

// ─── Confirmations ───────────────────────────────────────────────────────────

func (a App) confirmPanel() string {
	width := min(a.panelWidth(), 52) // a question needs less room than a listing
	keys := overlayStyle.Render("  y") + overlayDimStyle.Render(" yes    ") +
		overlayStyle.Render("n") + overlayDimStyle.Render(" no")
	body := []string{
		"",
		overlayStyle.Render("  " + a.confirm.message),
		"",
		keys,
		"",
	}
	return panelBox("", "", body, width)
}
