# The recording theme

The default theme, screen, paints its chrome in pure ANSI (0-15) rather than
hardcoded hex, precisely so it adopts whatever palette the terminal already
has. A recording is a terminal like any other, so this theme block *is* what
screen looks like here — not decoration, the actual colours eight of the
sixteen slots resolve to:

```
black        #000000   text on an accent fill (cursor glyph, chip text)
white        #C0C0C0   main text — the page, panel bodies (= foreground)
brightBlack  #808080   secondary text, and the resting "F1 Help" chip
cyan         #00A0A0   the accent: cursor fill, panel titles, borders
brightCyan   #00FFFF   the "F1 Help" chip's text
blue         #0000C0   the version chip's background
brightGreen  #00FF00   a success message
brightRed    #FF0000   an error message
```

The rest of the sixteen slots are unused — justwrite's other themes (paper,
mono) never reach for bare ANSI, and nothing on screen needs a plain red,
green, yellow, or magenta — filled in only because VHS wants a complete
palette and a half-filled one would leave them to chance.

Keep the block identical across tapes. If it changes, every capture has to be
regenerated together or the set stops looking like one product. If screen's
own palette in internal/ui/palette.go changes which ANSI slots it uses, this
block and its table above need to change with it.
