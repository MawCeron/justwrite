# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

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

- `Shift+Page Up`/`Shift+Page Down` extend the selection.

### Fixed

- Saves go through a unique temp file, preserve the original file's
  permission bits, and are fsynced before the rename into place.

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

[Unreleased]: https://github.com/MawCeron/justwrite/compare/v0.1.3...develop
[0.1.3]: https://github.com/MawCeron/justwrite/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/MawCeron/justwrite/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/MawCeron/justwrite/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/MawCeron/justwrite/releases/tag/v0.1.0
