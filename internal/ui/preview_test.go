package ui

import (
	"fmt"
	"os"
	"testing"

	"github.com/MawCeron/justwrite/internal/editor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestPreview is a hand-run frame dump: go test -run TestPreview -v
func TestPreview(t *testing.T) {
	if os.Getenv("PREVIEW") == "" {
		t.Skip("set PREVIEW=1")
	}
	// go test hands the binary a pipe, so lipgloss would drop every colour.
	// Force them on: the point of looking is to look at the colours.
	lipgloss.SetColorProfile(termenv.TrueColor)
	sample := "El cursor está aquí y nada más existe. " +
		"Esta línea es lo bastante larga como para que el ajuste por palabra tenga que partirla en varias filas de la página, que nunca pasa de 82 columnas por más ancha que sea la terminal.\n\nSegundo párrafo."

	frame := func(label string, build func(App) App) {
		a := App{ed: newEditorWith(sample), w: 80, h: 20}
		a.name = testApp(t).name
		a = build(a)
		fmt.Printf("\n=== %s ===\n%s\n", label, a.View())
	}

	frame("writing", func(a App) App { a.ed.Path = "borrador.md"; a.ed.Modified = true; return a })
	frame("saved message", func(a App) App { a.ed.Path = "borrador.md"; a.status = "saved"; return a })
	frame("save failed", func(a App) App {
		a.ed.Path = "borrador.md"
		a.status, a.statusErr = "open /ro/borrador.md: permission denied", true
		return a
	})
	frame("help", func(a App) App { a.mode = ModeHelp; return a })
	frame("about", func(a App) App { a.mode = ModeAbout; return a })
	frame("stats", func(a App) App { a.mode = ModeStats; return a })
	frame("confirm", func(a App) App {
		a.mode = ModeConfirm
		a.confirm = confirmation{message: "discard changes?"}
		return a
	})
	frame("open", func(a App) App {
		a.mode = ModeOpen
		a.exp.refresh(".", "")
		return a
	})
	frame("save as", func(a App) App {
		a.mode = ModeSaveAs
		a.onName = true
		a.exp.refresh(".", "")
		a.name.SetValue("capitulo-01.md")
		return a
	})

	// The floor: a tmux split and a terminal below the minimum.
	narrow := App{ed: newEditorWith(sample), w: 60, h: 18, name: testApp(t).name}
	fmt.Printf("\n=== 60 columns ===\n%s\n", narrow.View())
	tiny := App{ed: newEditorWith(sample), w: 24, h: 8, name: testApp(t).name}
	fmt.Printf("\n=== 24x8 ===\n%s\n", tiny.View())

	// And a key actually landing, through the real Update path.
	a, _ := press(testApp(t), tea.KeyMsg{Type: tea.KeyF1})
	fmt.Printf("\n=== f1 via Update ===\n%s\n", a.View())
}

func newEditorWith(s string) *editor.Editor {
	e := editor.New()
	e.SetText(s)
	return e
}
