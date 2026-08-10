# Recording the captures

The screens in `assets/` are generated, not hand-made. [VHS](https://github.com/charmbracelet/vhs)
drives a real PTY through a headless terminal, so what lands in the file is
exactly what a user would see, and regenerating after a UI change is one command.

## Prerequisites

VHS needs a real PTY, so this runs on Linux or macOS — on Windows, inside WSL,
not PowerShell.

```bash
# Debian / Ubuntu / WSL
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://repo.charm.sh/apt/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/charm.gpg
echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" \
  | sudo tee /etc/apt/sources.list.d/charm.list
sudo apt update && sudo apt install -y vhs ttyd ffmpeg gifsicle
```

```bash
# macOS
brew install vhs gifsicle
```

`ttyd` and `ffmpeg` do the recording and encoding. `gifsicle` is not optional:
VHS writes every frame in full, so the raw GIF is 10-20 MB and `gifsicle -O3`
takes it to well under a megabyte without touching a single pixel.

## Recording

```bash
./assets/tapes/record.sh
```

It cross-compiles a Linux binary into `assets/tapes/bin/` (ignored by git), runs
every tape, and optimizes the GIF. Go only has to be on the machine that builds;
the recording box just needs VHS.

## What each tape captures

| Tape | Output | Scene |
|---|---|---|
| `hero.tape` | `justwrite.gif` | Writing a couple of lines, then naming the file. The bar goes from empty, to `*`, to `saved`, to the filename |
| `dialog.tape` | `dialog.png`, `dialog-filter.png` | The open dialog, then the same listing narrowed by typing, with the counter in the bottom border |
| `stats.tape` | `stats.png` | The stats panel over a page with real prose behind it |

The theme is pinned in every tape — see `_theme.md` for what it controls and why
it has to stay identical across all of them.

## The help panel cannot be captured yet

VHS has no token for function keys — there is no `F1` in its keyword table — so
there is no way to open the help or About panels from a tape. Three ways out,
none of them taken yet:

- Give help a second binding VHS can reach, such as `Ctrl+G`. This has value of
  its own: several terminal emulators swallow `F1` for their own help, and it
  does not always survive SSH, so some users cannot open the panel at all today.
- Leave help out of the captures and let the README table document it.
- Send `ESC O P` raw from the tape to imitate F1. It depends on the three bytes
  arriving together; if they split, `OP` gets typed into the document and the
  screenshot comes out with rubbish in it.

## Wiring the captures into the README

Once the files exist, the ASCII mock in the main README is meant to be replaced:

```markdown
<img src="assets/justwrite.gif" alt="Writing in justwrite, then saving it" width="100%">
```

Leave the mock in place until then — a README pointing at images that are not
there yet looks worse than one with no images at all.
