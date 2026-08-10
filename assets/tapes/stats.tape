# Still: the stats panel over a page with real prose behind it.
#
# The numbers have to come from somewhere, so the text is typed off camera
# first — otherwise the panel reports a document of four words.

Set Shell "bash"
Set FontSize 20
Set Padding 20
Set CursorBlink false
Set TypingSpeed 8ms

# See _theme.md. Only the page background and default ink come from here.
Set Theme { "name": "justwrite", "background": "#0D0D0B", "foreground": "#E6E2D6", "selection": "#4A3A12", "cursor": "#F4D33E", "black": "#1A1815", "red": "#C25B5B", "green": "#7FA86B", "yellow": "#DD9803", "blue": "#6B8CA8", "magenta": "#A87FA8", "cyan": "#6BA8A0", "white": "#D7D3C6", "brightBlack": "#3A3222", "brightRed": "#FFB4B4", "brightGreen": "#A8E6A8", "brightYellow": "#F4D33E", "brightBlue": "#9BB8CE", "brightMagenta": "#CEA8CE", "brightCyan": "#9BCEC8", "brightWhite": "#F2EFE4" }

Hide
Type "export PATH=$PWD/assets/tapes/bin:$PATH" Enter
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
