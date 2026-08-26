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

const usage = `justwrite is a minimal, distraction-free terminal text editor.

Usage:
  justwrite              open a new document
  justwrite <file>       open a file, or start one under that name
  justwrite --version    print the version and exit
  justwrite --help       show this help`

// cli is a decision about what to do with the command line, kept separate
// from main so it can be tested without a process to exit or a terminal to
// take over.
type cli struct {
	message string // printed and exits before the editor opens, if not ""
	isError bool   // message goes to stderr with exit code 1, instead of stdout with 0
	path    string // the file to open, once message is empty
}

// parseArgs reads args (os.Args without the program name). --version and
// --help print and exit rather than opening a document named literally
// that; anything past the first path is rejected instead of silently
// dropped.
func parseArgs(args []string) cli {
	if len(args) == 0 {
		return cli{}
	}
	switch args[0] {
	case "--version":
		return cli{message: ui.VersionString()}
	case "--help":
		return cli{message: usage}
	}
	if len(args) > 1 {
		return cli{message: "justwrite: too many arguments — pass at most one file", isError: true}
	}
	return cli{path: args[0]}
}

func main() {
	result := parseArgs(os.Args[1:])
	if result.message != "" {
		out, code := os.Stdout, 0
		if result.isError {
			out, code = os.Stderr, 1
		}
		fmt.Fprintln(out, result.message)
		os.Exit(code)
	}

	// A file that cannot be read is reported here, while the terminal is still
	// the terminal — not swallowed into an empty document once the editor owns
	// the screen.
	app, err := ui.NewApp(result.path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "justwrite: %v\n", err)
		os.Exit(1)
	}

	if _, err := tea.NewProgram(app, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "justwrite: %v\n", err)
		os.Exit(1)
	}
}
