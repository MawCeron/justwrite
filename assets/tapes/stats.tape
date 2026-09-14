# Still: the stats panel over a page with real prose behind it.
#
# The numbers have to come from somewhere, so the text is typed off camera
# first — otherwise the panel reports a document of four words.

Set Shell "bash"
Set FontSize 20
Set Padding 20
Set CursorBlink false
Set TypingSpeed 8ms

# See _theme.md — screen's pure ANSI colours resolve from these ANSI slots.
Set Theme { "name": "justwrite", "background": "#0B0D0E", "foreground": "#C0C0C0", "selection": "#404040", "cursor": "#00A0A0", "black": "#000000", "red": "#C25B5B", "green": "#4A9078", "yellow": "#B08D57", "blue": "#0000C0", "magenta": "#8B7AA8", "cyan": "#00A0A0", "white": "#C0C0C0", "brightBlack": "#808080", "brightRed": "#FF0000", "brightGreen": "#00FF00", "brightYellow": "#D9B166", "brightBlue": "#6B8CFF", "brightMagenta": "#C79AE0", "brightCyan": "#00FFFF", "brightWhite": "#F0F0F0" }

# Now that screen resolves real colour from these slots instead of just
# completeness filler, give the browser a moment to actually apply the
# theme before anything paints — skipping this risks a flaky chip colour.
Sleep 300ms

Hide
Type "export PATH=$PWD/assets/tapes/bin:$PATH" Enter
# Otherwise justwrite reads whatever session/document goal the recording
# machine actually has set, and the panel comes out showing somebody's
# real numbers instead of a fresh install's.
Type "export XDG_CONFIG_HOME=$(mktemp -d)" Enter
Type "cd $(mktemp -d) && clear" Enter

Type "justwrite" Enter
Sleep 2s
Type "There is a kind of writing that only happens when nothing on the screen is asking for anything. No sidebar keeping count of the words as they arrive, no ribbon of buttons for things nobody presses, no red underline disagreeing with a name it has never seen before."
Enter 2
Type "The count is still there when you want it, one keystroke away, and gone again the moment you stop asking."
Sleep 600ms
Show

Ctrl+T
# A still gets one frame and no second chance: settle generously.
Sleep 2.5s
Screenshot assets/stats.png
