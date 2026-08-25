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

// The shortcuts, in the order they are worth learning.
var shortcuts = [][2]string{
	{"ctrl+s", "save"},
	{"ctrl+o", "open"},
	{"ctrl+n", "new"},
	{"ctrl+q", "quit"},
	{"ctrl+z", "undo"},
	{"ctrl+y", "redo"},
	{"ctrl+a", "select all"},
	{"ctrl+c", "copy"},
	{"ctrl+x", "cut"},
	{"ctrl+v", "paste"},
	{"ctrl+t", "stats"},
	{"ctrl+←→", "jump word"},
	{"shift+↑↓←→", "select"},
	{"pgup/pgdn", "scroll"},
	{"shift+pgup/pgdn", "select"},
	{"ctrl+home/end", "start/end"},
	{"ctrl+shift+home/end", "select"},
	{"tab", "insert 4 spaces"},
	{"esc", "close panel"},
}

func (a App) helpPanel() string {
	width := a.panelWidth()

	// helpGap is the breathing room after the longest key in a column — more
	// than the one space alignment alone requires, so the table reads as a
	// table rather than a packed list.
	const helpGap = 2

	cell := func(s [2]string, keyWidth int) string {
		gap := max(keyWidth-ansi.StringWidth(s[0]), 0) + helpGap
		// The indent is inside the styled run: two bare spaces would show the
		// terminal's own background as a notch in the panel.
		return overlayStyle.Render("  "+s[0]) + overlayDimStyle.Render(strings.Repeat(" ", gap)+s[1])
	}

	// keyWidth is the longest key in list; cellWidth is the widest rendered
	// cell that column will ever draw, both computed rather than guessed, so
	// a longer shortcut can only widen its own column instead of silently
	// misaligning or overflowing it.
	keyWidth := func(list [][2]string) int {
		w := 0
		for _, s := range list {
			w = max(w, ansi.StringWidth(s[0]))
		}
		return w
	}
	cellWidth := func(list [][2]string, kw int) int {
		w := 0
		for _, s := range list {
			w = max(w, 2+kw+helpGap+ansi.StringWidth(s[1]))
		}
		return w
	}

	rows := (len(shortcuts) + 1) / 2
	left, right := shortcuts[:rows], shortcuts[rows:]
	leftKey, rightKey := keyWidth(left), keyWidth(right)
	leftWidth, rightWidth := cellWidth(left, leftKey), cellWidth(right, rightKey)

	var body []string
	body = append(body, "")

	// Two columns only when both actually fit at their natural width — the
	// short "ctrl+X" column and the longer movement-key column need
	// different amounts of room, not an even half each.
	if width >= leftWidth+rightWidth {
		for i, s := range left {
			line := fitCell(cell(s, leftKey), leftWidth, overlayStyle.Render)
			if i < len(right) {
				line += cell(right[i], rightKey)
			}
			body = append(body, line)
		}
	} else {
		kw := max(leftKey, rightKey)
		for _, s := range shortcuts {
			body = append(body, cell(s, kw))
		}
	}

	body = append(body, "")

	// On a terminal too short for the whole list, say so rather than pretend
	// these are all the keys there are.
	footer := "? about"
	if maxRows := a.panelHeight(); len(body) > maxRows {
		body = body[:maxRows]
		footer = "…  " + footer
	}
	return panelBox("help", footer, body, width)
}

// ─── About ───────────────────────────────────────────────────────────────────

func (a App) aboutPanel() string {
	width := a.panelWidth()
	v, commit := buildInfo()
	title := appName + " " + v
	if commit != "" {
		title += " (" + commit + ")"
	}

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
