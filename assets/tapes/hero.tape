# The hero: writing something, then giving it a name.
#
# One story only — you type and the page is just text, and the one line at the
# foot keeps up: nothing, then *, then the filename. Everything else justwrite
# does is a still, not a scene.

Output assets/justwrite.gif

Set Shell "bash"
Set FontSize 20
Set Padding 20
Set CursorBlink false
Set TypingSpeed 45ms

# See _theme.md. Only the page background and default ink come from here.
Set Theme { "name": "justwrite", "background": "#0D0D0B", "foreground": "#E6E2D6", "selection": "#4A3A12", "cursor": "#F4D33E", "black": "#1A1815", "red": "#C25B5B", "green": "#7FA86B", "yellow": "#DD9803", "blue": "#6B8CA8", "magenta": "#A87FA8", "cyan": "#6BA8A0", "white": "#D7D3C6", "brightBlack": "#3A3222", "brightRed": "#FFB4B4", "brightGreen": "#A8E6A8", "brightYellow": "#F4D33E", "brightBlue": "#9BB8CE", "brightMagenta": "#CEA8CE", "brightCyan": "#9BCEC8", "brightWhite": "#F2EFE4" }

Hide
# The spawned shell inherits nothing, so PATH is exported here — unquoted, or
# bash would take $PATH literally and lose every command.
Type "export PATH=$PWD/assets/tapes/bin:$PATH" Enter
Type "cd $(mktemp -d) && clear" Enter
Show

Type "justwrite" Enter
Sleep 1.5s

# The bar starts empty: a new document has no name and nothing to lose yet.
Type "The cursor is here and nothing else exists."
Sleep 600ms
Enter 2

# The * arrives on the first keystroke and stays.
Type "No menus, no toolbars, no frame around the page."
Sleep 2s

# No name yet, so save asks for one, with the focus already in the field.
Ctrl+S
Sleep 1.6s
Type "draft.md"
Sleep 700ms
Enter

# "saved" in green, then it fades and leaves the filename behind.
Sleep 3s
