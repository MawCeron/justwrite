package ui

import (
	"slices"

	"github.com/charmbracelet/lipgloss"
)

// palette is every colour justwrite paints with. A theme is just a value of
// this type — swapping themes means picking a different palette and
// rebuilding the styles below from it, nothing more.
type palette struct {
	ink      lipgloss.Color // main text: the page, panel bodies
	quiet    lipgloss.Color // secondary text: labels, metadata
	onAccent lipgloss.Color // text drawn on top of an accent fill (the cursor glyph, the version chip)

	accent     lipgloss.Color // cursor background, panel titles, the selected folder in a listing
	accentText lipgloss.Color // the "F1 Help" chip's text, unselected folders — a secondary accent role
	border     lipgloss.Color // panel border lines
	versionBg  lipgloss.Color // the version chip's background

	barBg      lipgloss.Color // status bar background
	helpChipBg lipgloss.Color // "F1 Help" chip background, its resting state

	okFg, okBg   lipgloss.Color
	errFg, errBg lipgloss.Color

	overlayBg lipgloss.Color // every floating panel's background
	selectBg  lipgloss.Color // selected text, and the highlighted row in a listing

	// pageBg is the page's own background. Empty means none: the page stays
	// whatever the terminal already is.
	pageBg lipgloss.Color
}

// screen is pure ANSI (0-15): the sixteen colours the terminal itself
// defines, so the theme adopts whatever palette the user already has
// configured there rather than imposing one of its own. The hex values in
// the comments are only what those numbers typically render as — never
// hardcoded, that would defeat the point.
var screen = palette{
	ink: lipgloss.Color("7"), quiet: lipgloss.Color("8"), onAccent: lipgloss.Color("0"),

	accent: lipgloss.Color("6"), accentText: lipgloss.Color("14"),
	border: lipgloss.Color("6"), versionBg: lipgloss.Color("4"),

	barBg: lipgloss.Color("0"), helpChipBg: lipgloss.Color("8"),

	okFg: lipgloss.Color("10"), okBg: lipgloss.Color("0"),
	errFg: lipgloss.Color("9"), errBg: lipgloss.Color("0"),

	overlayBg: lipgloss.Color("0"), selectBg: lipgloss.Color("8"),
}

// paper is dark ink on an actual light page — not a nicety, but what makes
// justwrite usable on an e-ink writerdeck or in direct sunlight, where a dark
// page is unreadable regardless of what the terminal's own background is set
// to. The accent is a petrol blue rather than screen's cyan: it evokes ink
// rather than a terminal, and reads calmly on a light page.
var paper = palette{
	ink: lipgloss.Color("#2B2A28"), quiet: lipgloss.Color("#7A756C"), onAccent: lipgloss.Color("#FFFFFF"),

	accent: lipgloss.Color("#3B5B73"), accentText: lipgloss.Color("#4A7085"),
	border: lipgloss.Color("#5C7D8C"), versionBg: lipgloss.Color("#2F4A5C"),

	barBg: lipgloss.Color("#E4DFD2"), helpChipBg: lipgloss.Color("#D8D0BE"),

	okFg: lipgloss.Color("#1E4620"), okBg: lipgloss.Color("#D3E8D3"),
	errFg: lipgloss.Color("#7A2020"), errBg: lipgloss.Color("#F2D9D9"),

	overlayBg: lipgloss.Color("#EFEAE0"), selectBg: lipgloss.Color("#C7D6DC"),

	pageBg: lipgloss.Color("#F7F3EA"),
}

// mono sticks to grayscale ANSI for a terminal that cannot do truecolor and
// should not be asked to guess at colour either: everything is white, gray,
// or black, with one deliberate exception — the cursor, the single element
// that gets bright-white-on-black, a visual shout precisely because nothing
// else here is shouting. Error inverts (black on white) rather than reaching
// for a colour it does not have to mark itself out.
var mono = palette{
	ink: lipgloss.Color("7"), quiet: lipgloss.Color("8"), onAccent: lipgloss.Color("0"),

	accent: lipgloss.Color("15"), accentText: lipgloss.Color("8"),
	border: lipgloss.Color("8"), versionBg: lipgloss.Color("8"),

	barBg: lipgloss.Color("0"), helpChipBg: lipgloss.Color("0"),

	okFg: lipgloss.Color("7"), okBg: lipgloss.Color("0"),
	errFg: lipgloss.Color("0"), errBg: lipgloss.Color("7"),

	overlayBg: lipgloss.Color("0"), selectBg: lipgloss.Color("8"),
}

// themes is every theme a name can resolve to, and themeOrder is the order
// cycling steps through — a map alone would not have a stable one.
var themes = map[string]palette{
	"screen": screen,
	"paper":  paper,
	"mono":   mono,
}

var themeOrder = []string{"screen", "paper", "mono"}

// cycleTheme steps from current by delta (1 or -1), wrapping around. An
// unrecognized current — never set, or a stale name from an older build —
// starts from screen rather than erroring.
func cycleTheme(current string, delta int) string {
	i := max(slices.Index(themeOrder, current), 0)
	n := len(themeOrder)
	return themeOrder[((i+delta)%n+n)%n]
}

// hasPageBg and pageFillStyle let render.go paint the page's own background
// (margins, blank rows, the space beyond a short line) only when the active
// theme sets one — screen and mono leave the terminal's background alone, as
// justwrite always has.
var (
	hasPageBg     bool
	pageFillStyle lipgloss.Style
)

var (
	textStyle   lipgloss.Style
	dimStyle    lipgloss.Style
	cursorStyle lipgloss.Style
	selectStyle lipgloss.Style

	// fieldCursorStyle is cursorStyle's colours swapped: bubbles' cursor.Model
	// renders its Style with Reverse(true) applied on top, so handing it
	// cursorStyle as-is would flip it right back to plain text — this is what
	// actually comes out looking like cursorStyle once reversed.
	fieldCursorStyle lipgloss.Style

	versionStyle  lipgloss.Style
	noteStyle     lipgloss.Style
	helpChipStyle lipgloss.Style

	okNoteStyle  lipgloss.Style
	okChipStyle  lipgloss.Style
	errNoteStyle lipgloss.Style
	errChipStyle lipgloss.Style

	overlayStyle       lipgloss.Style
	overlayDimStyle    lipgloss.Style
	overlayBorderStyle lipgloss.Style
	overlayTitleStyle  lipgloss.Style
	overlayDirStyle    lipgloss.Style
	overlaySelStyle    lipgloss.Style
	overlaySelDirStyle lipgloss.Style
)

// applyTheme rebuilds every style above from the named theme. An unknown
// name falls back to screen without complaining — NO_COLOR keeps winning
// over all of this regardless, since it is termenv underneath that decides
// whether any of these colours actually get sent to the terminal.
func applyTheme(name string) {
	p, ok := themes[name]
	if !ok {
		p = screen
	}

	textStyle = lipgloss.NewStyle().Foreground(p.ink)
	dimStyle = lipgloss.NewStyle().Foreground(p.quiet)
	cursorStyle = lipgloss.NewStyle().Foreground(p.onAccent).Background(p.accent)
	selectStyle = lipgloss.NewStyle().Foreground(p.ink).Background(p.selectBg)

	fieldCursorStyle = lipgloss.NewStyle().Foreground(p.accent).Background(p.onAccent)

	versionStyle = lipgloss.NewStyle().Foreground(p.onAccent).Background(p.versionBg).Bold(true)
	noteStyle = lipgloss.NewStyle().Foreground(p.quiet).Background(p.barBg)
	helpChipStyle = lipgloss.NewStyle().Foreground(p.accentText).Background(p.helpChipBg)

	okNoteStyle = lipgloss.NewStyle().Foreground(p.okFg).Background(p.okBg)
	okChipStyle = lipgloss.NewStyle().Foreground(p.okBg).Background(p.okFg)
	errNoteStyle = lipgloss.NewStyle().Foreground(p.errFg).Background(p.errBg)
	errChipStyle = lipgloss.NewStyle().Foreground(p.errBg).Background(p.errFg)

	overlayStyle = lipgloss.NewStyle().Foreground(p.ink).Background(p.overlayBg)
	overlayDimStyle = lipgloss.NewStyle().Foreground(p.quiet).Background(p.overlayBg)
	overlayBorderStyle = lipgloss.NewStyle().Foreground(p.border).Background(p.overlayBg)
	overlayTitleStyle = lipgloss.NewStyle().Foreground(p.accent).Background(p.overlayBg).Bold(true)
	overlayDirStyle = lipgloss.NewStyle().Foreground(p.accentText).Background(p.overlayBg)
	overlaySelStyle = lipgloss.NewStyle().Foreground(p.ink).Background(p.selectBg)
	overlaySelDirStyle = lipgloss.NewStyle().Foreground(p.accent).Background(p.selectBg)

	hasPageBg = p.pageBg != ""
	if hasPageBg {
		textStyle = textStyle.Background(p.pageBg)
		pageFillStyle = lipgloss.NewStyle().Background(p.pageBg)
	} else {
		pageFillStyle = lipgloss.NewStyle()
	}
}

func init() { applyTheme("screen") }
