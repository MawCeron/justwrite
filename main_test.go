package main

import (
	"testing"

	"github.com/MawCeron/justwrite/internal/ui"
)

func TestParseArgs(t *testing.T) {
	if got := parseArgs(nil); got.message != "" || got.isError || got.path != "" {
		t.Errorf("no args: %+v, want a new document", got)
	}

	if got := parseArgs([]string{"draft.md"}); got.message != "" || got.path != "draft.md" {
		t.Errorf("one file: %+v, want path %q", got, "draft.md")
	}

	if got := parseArgs([]string{"a.md", "b.md"}); !got.isError || got.message == "" {
		t.Errorf("two files: %+v, want a rejecting error", got)
	}
}

// --version and --help must print and exit, not open a document literally
// named "--version" or "--help".
func TestVersionAndHelpDoNotOpenADocument(t *testing.T) {
	if got := parseArgs([]string{"--version"}); got.path != "" || got.isError || got.message != ui.VersionString() {
		t.Errorf("--version: %+v", got)
	}

	if got := parseArgs([]string{"--help"}); got.path != "" || got.isError || got.message != usage {
		t.Errorf("--help: %+v", got)
	}
}
