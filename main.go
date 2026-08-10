// justwrite is a minimal, distraction-free terminal text editor.
//
//	justwrite            # a new document
//	justwrite notes.md   # open a file, or start one under that name
package main

import (
	"fmt"
	"os"

	"github.com/MawCeron/justwrite/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	path := ""
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	// A file that cannot be read is reported here, while the terminal is still
	// the terminal — not swallowed into an empty document once the editor owns
	// the screen.
	app, err := ui.NewApp(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "justwrite: %v\n", err)
		os.Exit(1)
	}

	if _, err := tea.NewProgram(app, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "justwrite: %v\n", err)
		os.Exit(1)
	}
}
