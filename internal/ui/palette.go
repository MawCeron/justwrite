package ui

import "github.com/charmbracelet/lipgloss"

// Amber Trails — https://www.schemecolor.com/amber-trails.php
var (
	goldYellow  = lipgloss.Color("#F4D33E")
	darkGold    = lipgloss.Color("#F1BC1B")
	berlin      = lipgloss.Color("#E2B01C")
	harvestGold = lipgloss.Color("#DD9803")
)

// Every colour in justwrite lives here, so any of them is a one-line change.
//
// The chrome is amber and it is meant to be legible in a badly lit room: the
// chips and borders carry real contrast rather than whispering at the edge of
// the background. Only the chrome is warm — the page itself stays off-white on
// black, because a page tinted gold is a page you cannot read for an hour.
//
// NO_COLOR is honoured for free: lipgloss degrades to plain text through
// termenv, and the layout carries the meaning on its own.
var (
	ink    = lipgloss.Color("#E6E2D6") // the writing
	quiet  = lipgloss.Color("#B0A794") // labels and metadata, still readable
	shadow = lipgloss.Color("#14140F") // text laid on top of an amber fill

	barBg      = lipgloss.Color("#1C1C1A")
	helpChipBg = lipgloss.Color("#3A3222") // amber-tinted, so the hint reads as one

	okFg, okBg   = lipgloss.Color("#A8E6A8"), lipgloss.Color("#1E4620")
	errFg, errBg = lipgloss.Color("#FFB4B4"), lipgloss.Color("#5A1E1E")

	overlayBg = lipgloss.Color("#1A1815")
	selectBg  = lipgloss.Color("#4A3A12") // an amber wash, distinct from the cursor
)

var (
	textStyle   = lipgloss.NewStyle().Foreground(ink)
	dimStyle    = lipgloss.NewStyle().Foreground(quiet)
	cursorStyle = lipgloss.NewStyle().Foreground(shadow).Background(goldYellow)
	selectStyle = lipgloss.NewStyle().Foreground(ink).Background(selectBg)

	// fieldCursorStyle is cursorStyle's colours swapped: bubbles' cursor.Model
	// renders its Style with Reverse(true) applied on top, so handing it
	// cursorStyle as-is would flip it right back to plain text on black —
	// this is what actually comes out looking like cursorStyle once reversed.
	fieldCursorStyle = lipgloss.NewStyle().Foreground(goldYellow).Background(shadow)

	versionStyle  = lipgloss.NewStyle().Foreground(shadow).Background(harvestGold).Bold(true)
	noteStyle     = lipgloss.NewStyle().Foreground(quiet).Background(barBg)
	helpChipStyle = lipgloss.NewStyle().Foreground(darkGold).Background(helpChipBg)

	okNoteStyle  = lipgloss.NewStyle().Foreground(okFg).Background(okBg)
	okChipStyle  = lipgloss.NewStyle().Foreground(okBg).Background(okFg)
	errNoteStyle = lipgloss.NewStyle().Foreground(errFg).Background(errBg)
	errChipStyle = lipgloss.NewStyle().Foreground(errBg).Background(errFg)

	overlayStyle       = lipgloss.NewStyle().Foreground(ink).Background(overlayBg)
	overlayDimStyle    = lipgloss.NewStyle().Foreground(quiet).Background(overlayBg)
	overlayBorderStyle = lipgloss.NewStyle().Foreground(berlin).Background(overlayBg)
	overlayTitleStyle  = lipgloss.NewStyle().Foreground(goldYellow).Background(overlayBg).Bold(true)
	overlayDirStyle    = lipgloss.NewStyle().Foreground(darkGold).Background(overlayBg)
	overlaySelStyle    = lipgloss.NewStyle().Foreground(ink).Background(selectBg)
	overlaySelDirStyle = lipgloss.NewStyle().Foreground(goldYellow).Background(selectBg)
)
