# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [1.0.0] - 2026-10-22

### Fixed

- The page is measured in terminal cells and grapheme clusters, not runes: CJK
  text, emoji (ZWJ sequences and flags included) and combining marks no longer
  overflow the page or throw off the cursor, the selection and the centring,
  and a row never starts with a mark cut off from its base.
- The cursor, selection, Backspace and Delete step over whole grapheme
  clusters, so a combining mark, flag or emoji family is never split.
- Up and Down keep the cursor's column in cells, not runes, across CJK text
  and tabs, and forget it after any edit, select all, or anything else that
  moves the cursor.
- Literal tabs in a loaded file are drawn as spaces to the next tab stop
  instead of spilling past the page. The file itself is not changed.
- A document that shrinks to fit the window scrolls back to its first row.
- Long documents wrap once per change instead of several times per keystroke.
- The open dialog appears before justwrite has looked inside the files to hide
  the binary ones, so a slow disk or network share no longer freezes it, and
  its highlight stays on the same file when binaries above it are hidden.
- Recovering autosaved text no longer reads as saved after an undo, which
  skipped the quit prompt and then deleted the only copy, and keeps the file's
  CRLF line endings.
- A file that is not valid UTF-8 is refused instead of being opened and
  rewritten with replacement characters on save.
- Pasted text has its line endings normalised, as a loaded file does, and its
  control characters (escape sequences included) removed.
- Saving under a new name removes the old name's autosave copy, and reloading
  from the conflict panel removes the discarded edits' copy.
- Saving a document opened through a symlink, even one whose target does not
  exist yet, writes to the target and keeps the link.

## [0.2.0] - 2026-09-15

### Added

- Find (`Ctrl+F`): case-insensitive, wraps around. `Enter`/`n` for the next
  match, `Shift+N` for the previous, `Tab` to edit the query again without
  closing, `Esc` closes and restores the cursor.
- `Ctrl+Home`/`Ctrl+End` jump to the start or end of the document;
  `Ctrl+Shift+Home/End` extends the selection there.
- `Ctrl+W` deletes the word before the cursor.
- Session and document word goals, set from the stats panel (`g`/`d`) and
  shown as `current / goal`.
- `--version` and `--help` command-line flags.
- The editor's cursor, and every text-field cursor, now blinks.
- Three themes — screen (pure ANSI, the default), paper (a real light page),
  and mono (grayscale ANSI) — cycled with `←`/`→` from the help panel and
  saved to the config file.
- Reopening a document — saving it, switching to another through the open
  dialog, or launching justwrite with its path again — puts the cursor back
  where it was left, instead of always at the end.
- `Ctrl+R` in the open dialog lists the ten most recently opened files.
- Autosave and crash recovery: unsaved changes are written to a hidden swap
  file every few seconds, and offered back the next time the document opens
  if the swap is newer than it.

### Changed

- The help panel is now scrollable, two lines per shortcut, sized to fit its
  own content instead of a fixed grid.
- Backspace groups into one undo step per burst instead of one per character.
- Page Up scrolls the viewport the same way Page Down already did.
- Vertical movement remembers the column you started from, even after
  passing through a shorter line.
- `Ctrl+S` refuses to overwrite a file that changed on disk since it was
  opened; a new panel offers reload, overwrite anyway, or save as instead.

### Fixed

- The status bar no longer hides the filename while a status message is
  showing.
- Undoing back to exactly the saved state clears the modified marker.
- Text-field cursors (save-as, find, stats goals) match the editor's own
  color instead of rendering dark.

## [0.1.3] - 2026-08-25

### Fixed

- The overwrite guard now compares absolute paths, so opening the same file
  by a relative and an absolute path is recognized as the same document.
- The save-as listing scrolls by the number of rows it actually draws.

## [0.1.2] - 2026-08-19

### Fixed

- CRLF line endings are normalized on load and restored on save, instead of
  being silently rewritten as LF.

## [0.1.1] - 2026-08-19

### Added

- `Shift+Page Up`/`Shift+Page Down` extend the selection
  ([@OdaloV](https://github.com/OdaloV)).

### Fixed

- Saves go through a unique temp file, preserve the original file's
  permission bits, and are fsynced before the rename into place.
- README claimed Go 1.24+ where go.mod actually requires 1.26
  ([@YuuGR1337](https://github.com/YuuGR1337)).

### Thanks

[@OdaloV](https://github.com/OdaloV) and [@YuuGR1337](https://github.com/YuuGR1337)
for their first contributions to justwrite.

## [0.1.0] - 2026-08-10

First release.

### Added

- A distraction-free terminal text editor built with Bubble Tea: write,
  save, undo/redo in bursts, select all, copy/cut/paste.
- Open and save-as file dialogs, with hidden and binary files filtered out
  by default.
- Stats panel: words, characters, pages, read time.
- Help (`F1`) and About (`?`) panels.
- Packages for Linux, macOS and Windows, x86-64 and ARM: `.deb`, `.rpm`,
  `.tar.gz`, `.zip`.

[Unreleased]: https://github.com/MawCeron/justwrite/compare/v1.0.0...develop
[1.0.0]: https://github.com/MawCeron/justwrite/compare/v0.2.0...v1.0.0
[0.2.0]: https://github.com/MawCeron/justwrite/compare/v0.1.3...v0.2.0
[0.1.3]: https://github.com/MawCeron/justwrite/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/MawCeron/justwrite/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/MawCeron/justwrite/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/MawCeron/justwrite/releases/tag/v0.1.0
