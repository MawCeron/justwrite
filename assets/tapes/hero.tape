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
