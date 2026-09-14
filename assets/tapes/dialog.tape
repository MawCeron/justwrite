# Stills: the open dialog, plain and then filtered.
#
# Two frames from one session — the listing as you meet it, and the same listing
# narrowed by typing, which is the only time the counter in the bottom border
# shows up.

Set Shell "bash"
Set FontSize 20
Set Padding 20
Set CursorBlink false
Set TypingSpeed 60ms

# See _theme.md — screen's pure ANSI colours resolve from these ANSI slots.
Set Theme { "name": "justwrite", "background": "#0B0D0E", "foreground": "#C0C0C0", "selection": "#404040", "cursor": "#00A0A0", "black": "#000000", "red": "#C25B5B", "green": "#4A9078", "yellow": "#B08D57", "blue": "#0000C0", "magenta": "#8B7AA8", "cyan": "#00A0A0", "white": "#C0C0C0", "brightBlack": "#808080", "brightRed": "#FF0000", "brightGreen": "#00FF00", "brightYellow": "#D9B166", "brightBlue": "#6B8CFF", "brightMagenta": "#C79AE0", "brightCyan": "#00FFFF", "brightWhite": "#F0F0F0" }

# Now that screen resolves real colour from these slots instead of just
# completeness filler, give the browser a moment to actually apply the
# theme before anything paints — skipping this risks a flaky chip colour.
Sleep 300ms

Hide
# The spawned shell inherits nothing, so PATH is exported here — unquoted, or
# bash would take $PATH literally and lose every command.
Type "export PATH=$PWD/assets/tapes/bin:$PATH" Enter

# A fixed path, not mktemp: the directory is on camera in the panel title, and a
# random /tmp/tmp.XmQ9 would change between runs. Nothing is deleted — the same
# files are simply made sure of.
Type "mkdir -p /tmp/justwrite-demo/notes && cd /tmp/justwrite-demo" Enter
Type "touch chapter-01.md chapter-02.md chapter-03.md outline.md research.txt" Enter
Type "clear" Enter

Type "justwrite chapter-01.md" Enter
Sleep 2s
Show

Ctrl+O
Sleep 2.5s
Screenshot assets/dialog.png

# Typing filters as you go; the counter appears once there is something to count.
Type "cha"
Sleep 2.5s
Screenshot assets/dialog-filter.png
