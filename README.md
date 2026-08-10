<div align="center">

# justwrite

A distraction-free terminal text editor. Black screen, a centered page, and one line at the foot that tells you where you are.

[![License](https://img.shields.io/github/license/MawCeron/justwrite?style=for-the-badge)](https://github.com/MawCeron/justwrite/blob/main/LICENSE)
![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Bubble Tea](https://img.shields.io/badge/Bubble%20Tea-FF69B4?style=for-the-badge)

```
                    The cursor is here and nothing else exists.
                    █



 v0.1.0  draft.md*                                              F1 Help
```

</div>

## What is this?

A text editor for writing prose, not code. There are no toolbars, no menus, no
line numbers and no frame around the page — just text, centered, never wider
than 82 columns because lines longer than that are hard to track back to.

Everything the editor knows about itself lives in a single line at the bottom:
the version, the document, and the way into the help. Every action is a keyboard
shortcut; you learn them once and they disappear.

It is one static binary with no cgo and no display server, so it runs anywhere
you can open a terminal — including a headless writerdeck.

## Quick Start

```bash
go install github.com/MawCeron/justwrite@latest
```

Or from a clone:

```bash
git clone https://github.com/MawCeron/justwrite
cd justwrite
go build .
```

Then:

```bash
justwrite              # a new document
justwrite draft.md     # open a file, or start one under that name
```

Naming a file that does not exist yet is how you start one: the name sticks, and
`Ctrl+S` saves straight to it.

On the machine you write on, that is all of it — `go build .` produces the
binary for the machine it ran on. `GOOS`/`GOARCH` only come into it when the
writing device is *not* the machine you build on, such as an ARM writerdeck
built from a laptop:

```bash
GOOS=linux GOARCH=arm64 go build .   # for an ARM device, from elsewhere
```

## Keyboard shortcuts

| Shortcut | Action |
|---|---|
| `Ctrl+S` | Save |
| `Ctrl+O` | Open file |
| `Ctrl+N` | New document |
| `Ctrl+Q` | Quit |
| `Ctrl+Z` / `Ctrl+Y` | Undo / redo |
| `Ctrl+A` | Select all |
| `Ctrl+C` / `Ctrl+X` / `Ctrl+V` | Copy / cut / paste |
| `Ctrl+T` | Stats — words, characters, pages, read time |
| `F1` | Help |
| `?` | About, from inside the help. Anywhere else it types a `?` |
| `Ctrl+←/→` | Jump a word left or right |
| `Shift+arrows` | Extend the selection |
| `Page Up/Down` | Scroll by page |
| `Tab` | Insert 4 spaces |
| `Esc` | Close panel / cancel |

Undo works in bursts: one `Ctrl+Z` takes back a word, not a letter.

### In the file dialogs

| Shortcut | Action |
|---|---|
| type | Filter the listing |
| `Ctrl+H` | Show hidden and binary files |
| `Tab` | Switch between the listing and the filename field |
| `Enter` | Enter a directory, or open the file |
| `Esc` | Clear the filter, then close |

Binary files stay out of the listing by default — opening a PNG only fills the
page with control characters. Stepping up to the parent directory leaves the
cursor on the directory you came out of.

Anything that would throw away unsaved work — quitting, starting a new document,
opening another file, saving over an existing one — asks first.

## The status bar

| Shows | Means |
|---|---|
| *(nothing)* | A new document, untouched |
| `*` | A new document with changes. There is no name yet, so it does not invent one |
| `draft.md` | Saved, and matching what is on disk |
| `draft.md*` | Edited since the last save |
| `saved` | Written to disk just now |
| a message in red | The save failed, and why. It stays until one succeeds |

## Details worth knowing

**Saving is atomic.** Writes go to a temporary file beside the document and are
renamed into place, so an interrupted or failed save leaves the previous draft
intact rather than a half-written file.

**The clipboard travels.** Copy and cut go through OSC 52, an escape sequence
the terminal emulator handles itself, so copying works over SSH and inside tmux
with no display server. For tmux, add `set -g set-clipboard on`. Pasting from
outside arrives as a bracketed paste and lands as a single undo step.

**Colours are optional.** `NO_COLOR` is honoured, and the layout carries the
meaning on its own.

## Project structure

```
justwrite/
├── internal/
│   ├── editor/    the document: buffer, selection, undo, wrapping, files
│   └── ui/        Bubble Tea model, status bar, overlays, file dialogs
├── LICENSE
├── README.md
├── go.mod
├── go.sum
└── main.go
```

`internal/editor` does not import Bubble Tea. It is plain logic over a rune
buffer, which is why most of the tests live there and run in milliseconds.

## Contributing

Issues and pull requests are welcome.

```bash
go test ./...   # the whole suite
go vet ./...
go build .
```

To look at the interface without a terminal, `PREVIEW=1 go test ./internal/ui -run TestPreview -v`
dumps every screen — page, status bar, help, about, stats, both file dialogs —
at several terminal sizes.

## License

[MIT](LICENSE) © 2026 Mauricio Cerón
