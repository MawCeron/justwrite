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

# See _theme.md. Only the page background and default ink come from here.
Set Theme { "name": "justwrite", "background": "#0D0D0B", "foreground": "#E6E2D6", "selection": "#4A3A12", "cursor": "#F4D33E", "black": "#1A1815", "red": "#C25B5B", "green": "#7FA86B", "yellow": "#DD9803", "blue": "#6B8CA8", "magenta": "#A87FA8", "cyan": "#6BA8A0", "white": "#D7D3C6", "brightBlack": "#3A3222", "brightRed": "#FFB4B4", "brightGreen": "#A8E6A8", "brightYellow": "#F4D33E", "brightBlue": "#9BB8CE", "brightMagenta": "#CEA8CE", "brightCyan": "#9BCEC8", "brightWhite": "#F2EFE4" }

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
