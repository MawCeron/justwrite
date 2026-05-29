# justwrite

A minimal, distraction-free terminal text editor. Black screen. A centered page. Nothing else.

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  The cursor is here and nothing else exists.            │
│  _                                                      │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## Philosophy

No toolbars. No status bars. No visible menus. Every action is a keyboard shortcut — you learn them once and they disappear. The only UI is the text itself, and you know, the basic needs (open, save, stats).

## Installation

Requires [Rust](https://rustup.rs/) 1.70 or later.

```bash
git clone https://github.com/MawCeron/justwrite
cd justwrite
cargo build --release
```

The binary will be at `target/release/justwrite`. Copy it anywhere in your `$PATH`.

### Optional: system clipboard

To enable copy/paste with the system clipboard (requires a display server):

```bash
cargo build --release --features system-clipboard
```

Without this flag, justwrite uses an internal clipboard that works in any terminal, including headless setups.

## Usage

```bash
justwrite                  # new document
justwrite myfile.txt       # open existing file
```

## Keyboard shortcuts

| Shortcut | Action |
|---|---|
| `Ctrl+S` | Save |
| `Ctrl+O` | Open file |
| `Ctrl+N` | New document |
| `Ctrl+Q` | Quit |
| `Ctrl+Z` | Undo |
| `Ctrl+Y` | Redo |
| `Ctrl+C` | Copy selection |
| `Ctrl+X` | Cut selection |
| `Ctrl+V` | Paste |
| `Ctrl+A` | Select all |
| `Ctrl+T` | Stats (words, characters, pages, read time) |
| `Ctrl+←/→` | Jump word left/right |
| `Shift+arrows` | Extend selection |
| `Shift+Home/End` | Extend selection to line start/end |
| `Page Up/Down` | Scroll by page |
| `Tab` | Insert 4 spaces |
| `Esc` | Close panel / cancel |

## Visual cues

The page border turns amber when there are unsaved changes. The terminal title shows the filename and a `*` when modified.

## Building for a writerdeck

justwrite is designed to run on a headless system with no display server. For a dedicated writing device:

```bash
cargo build --release
```

## License

MIT
