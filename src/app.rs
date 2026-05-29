use std::{fs, io, path::PathBuf};

use ratatui::widgets::ListState;

use crate::types::{EditOp, Mode, VisualLine};

// ─── App ──────────────────────────────────────────────────────────────────────

pub struct App {
    pub buffer:   Vec<char>,
    pub cursor:   usize,
    pub file_path: Option<PathBuf>,
    pub modified: bool,
    pub mode:     Mode,

    // Scroll
    pub scroll_offset: usize,

    // Selection: Some(anchor) means anchor..cursor is selected
    pub anchor: Option<usize>,

    // Internal clipboard
    pub clipboard: String,

    // Undo / redo stacks
    pub undo_stack: Vec<EditOp>,
    pub redo_stack: Vec<EditOp>,
    // Pending insert: accumulates a burst of typed chars before committing to undo stack
    pub pending_insert: Option<(usize, String)>, // (start_pos, accumulated_text)

    // File explorer
    pub explorer_entries: Vec<PathBuf>,
    pub explorer_state:   ListState,
    pub current_dir:      PathBuf,

    // Filename input (SaveAs)
    pub input: String,
    // When true, keyboard focus is on the filename input; when false, on the explorer
    pub save_as_input_focused: bool,
}

impl App {
    pub fn new() -> Self {
        let mut s = Self {
            buffer:   Vec::new(),
            cursor:   0,
            file_path: None,
            modified: false,
            mode:     Mode::Write,
            scroll_offset: 0,
            anchor:    None,
            clipboard: String::new(),
            undo_stack: Vec::new(),
            redo_stack: Vec::new(),
            pending_insert: None,
            explorer_entries: Vec::new(),
            explorer_state:   ListState::default(),
            current_dir: std::env::current_dir().unwrap_or_else(|_| PathBuf::from(".")),
            input: String::new(),
            save_as_input_focused: false,
        };
        s.refresh_explorer();
        s
    }

    pub fn open_file(path: PathBuf) -> io::Result<Self> {
        let content = fs::read_to_string(&path)?;
        let mut app = Self::new();
        app.buffer    = content.chars().collect();
        app.cursor    = app.buffer.len();
        app.file_path = Some(path);
        app.modified  = false;
        Ok(app)
    }

    /// Resets the editor to a blank document.
    pub fn new_document(&mut self) {
        self.buffer.clear();
        self.cursor       = 0;
        self.file_path    = None;
        self.modified     = false;
        self.scroll_offset = 0;
        self.anchor       = None;
        self.undo_stack.clear();
        self.redo_stack.clear();
        self.pending_insert = None;
    }

    // ─── Wrapping ─────────────────────────────────────────────────────────────

    pub fn visual_lines(&self, width: usize) -> Vec<VisualLine> {
        let mut lines = Vec::new();
        let buf = &self.buffer;
        let len = buf.len();

        if width == 0 { return lines; }

        let mut pos = 0usize;
        while pos <= len {
            if pos == len {
                lines.push(VisualLine { buf_start: pos, buf_end: pos, hard_break: false });
                break;
            }
            let newline_pos = buf[pos..].iter().position(|&c| c == '\n').map(|i| pos + i);
            let logical_end = newline_pos.unwrap_or(len);
            let mut seg_start = pos;

            loop {
                let seg_len = logical_end - seg_start;
                if seg_len <= width {
                    lines.push(VisualLine {
                        buf_start: seg_start,
                        buf_end:   logical_end,
                        hard_break: newline_pos.is_some(),
                    });
                    break;
                }
                let wrap_at = buf[seg_start..seg_start + width]
                    .iter().rposition(|&c| c == ' ')
                    .map(|i| i + 1)
                    .unwrap_or(width);
                lines.push(VisualLine { buf_start: seg_start, buf_end: seg_start + wrap_at, hard_break: false });
                seg_start += wrap_at;
            }

            pos = logical_end + if newline_pos.is_some() { 1 } else { len + 1 };
        }

        if lines.is_empty() {
            lines.push(VisualLine { buf_start: 0, buf_end: 0, hard_break: false });
        }
        lines
    }

    pub fn cursor_visual_line(&self, vlines: &[VisualLine]) -> usize {
        for (i, vl) in vlines.iter().enumerate() {
            if self.cursor >= vl.buf_start && self.cursor <= vl.buf_end {
                return i;
            }
        }
        vlines.len().saturating_sub(1)
    }

    pub fn adjust_scroll(&mut self, vlines: &[VisualLine], visible_height: usize) {
        let cursor_line = self.cursor_visual_line(vlines);
        let margin = 3usize;

        if cursor_line < self.scroll_offset + margin && self.scroll_offset > 0 {
            self.scroll_offset = cursor_line.saturating_sub(margin);
        }
        let bottom = self.scroll_offset + visible_height;
        if cursor_line + margin >= bottom {
            let new_offset = (cursor_line + margin + 1).saturating_sub(visible_height);
            self.scroll_offset = new_offset.min(vlines.len().saturating_sub(1));
        }
        if self.scroll_offset + visible_height > vlines.len() && vlines.len() >= visible_height {
            self.scroll_offset = vlines.len() - visible_height;
        }
    }

    // ─── Selection ────────────────────────────────────────────────────────────

    /// Ordered selection range (start, end).
    pub fn selection(&self) -> Option<(usize, usize)> {
        self.anchor.map(|a| {
            if a <= self.cursor { (a, self.cursor) } else { (self.cursor, a) }
        })
    }

    pub fn clear_selection(&mut self) {
        self.anchor = None;
    }

    pub fn start_selection(&mut self) {
        if self.anchor.is_none() {
            self.anchor = Some(self.cursor);
        }
    }

    /// Delete selected text and return it. Moves cursor to selection start.
    fn delete_selection(&mut self) -> Option<String> {
        let (start, end) = self.selection()?;
        let text: String = self.buffer[start..end].iter().collect();
        self.buffer.drain(start..end);
        self.cursor = start;
        self.anchor = None;
        self.modified = true;
        Some(text)
    }

    // ─── Clipboard ────────────────────────────────────────────────────────────

    pub fn copy(&mut self) {
        if let Some((start, end)) = self.selection() {
            let text: String = self.buffer[start..end].iter().collect();
            self.clipboard = text.clone();
            self.try_set_system_clipboard(&text);
        }
    }

    pub fn cut(&mut self) {
        self.commit_pending();
        if let Some((start, end)) = self.selection() {
            let text: String = self.buffer[start..end].iter().collect();
            self.clipboard = text.clone();
            self.try_set_system_clipboard(&text);
            // Record as delete for undo
            self.undo_stack.push(EditOp::Delete { at: start, text: text.clone() });
            self.redo_stack.clear();
            self.buffer.drain(start..end);
            self.cursor = start;
            self.anchor = None;
            self.modified = true;
        }
    }

    pub fn paste(&mut self) {
        self.commit_pending();
        // Try system clipboard first; fall back to internal
        let text = self.try_get_system_clipboard().unwrap_or_else(|| self.clipboard.clone());
        if text.is_empty() { return; }

        // Delete selection if any
        if self.selection().is_some() {
            let deleted = self.delete_selection().unwrap_or_default();
            if !deleted.is_empty() {
                self.undo_stack.push(EditOp::Delete { at: self.cursor, text: deleted });
            }
        }

        let at = self.cursor;
        for (i, c) in text.chars().enumerate() {
            self.buffer.insert(at + i, c);
        }
        self.cursor += text.chars().count();
        self.modified = true;
        self.undo_stack.push(EditOp::Insert { at, text });
        self.redo_stack.clear();
    }

    fn try_set_system_clipboard(&self, _text: &str) {
        #[cfg(feature = "system-clipboard")]
        {
            if let Ok(mut cb) = arboard::Clipboard::new() {
                let _ = cb.set_text(_text);
            }
        }
    }

    fn try_get_system_clipboard(&self) -> Option<String> {
        #[cfg(feature = "system-clipboard")]
        {
            if let Ok(mut cb) = arboard::Clipboard::new() {
                return cb.get_text().ok();
            }
        }
        None
    }

    // ─── Undo / Redo ──────────────────────────────────────────────────────────

    /// Flush any accumulated pending insert to the undo stack.
    /// Must be called before any operation that breaks the typing burst.
    pub fn commit_pending(&mut self) {
        if let Some((at, text)) = self.pending_insert.take() {
            if !text.is_empty() {
                self.undo_stack.push(EditOp::Insert { at, text });
            }
        }
    }

    pub fn undo(&mut self) {
        self.commit_pending();
        if let Some(op) = self.undo_stack.pop() {
            match &op {
                EditOp::Insert { at, text } => {
                    let at = *at;
                    let len = text.chars().count();
                    self.buffer.drain(at..at + len);
                    self.cursor = at;
                }
                EditOp::Delete { at, text } => {
                    let at = *at;
                    for (i, c) in text.chars().enumerate() {
                        self.buffer.insert(at + i, c);
                    }
                    self.cursor = at + text.chars().count();
                }
            }
            self.redo_stack.push(op);
            self.anchor = None;
            self.modified = true;
        }
    }

    pub fn redo(&mut self) {
        self.commit_pending();
        if let Some(op) = self.redo_stack.pop() {
            match &op {
                EditOp::Insert { at, text } => {
                    let at = *at;
                    for (i, c) in text.chars().enumerate() {
                        self.buffer.insert(at + i, c);
                    }
                    self.cursor = at + text.chars().count();
                }
                EditOp::Delete { at, text } => {
                    let at = *at;
                    let len = text.chars().count();
                    self.buffer.drain(at..at + len);
                    self.cursor = at;
                }
            }
            self.undo_stack.push(op);
            self.anchor = None;
            self.modified = true;
        }
    }

    // ─── Editing ──────────────────────────────────────────────────────────────

    pub fn insert_char(&mut self, c: char) {
        if self.selection().is_some() {
            self.commit_pending();
            let deleted = self.delete_selection().unwrap_or_default();
            if !deleted.is_empty() {
                self.undo_stack.push(EditOp::Delete { at: self.cursor, text: deleted });
            }
            self.redo_stack.clear();
        }
        // Commit if this char would break a contiguous burst:
        // - Enter/newline always gets its own op
        // - Space commits the previous word and starts fresh
        if c == '\n' {
            self.commit_pending();
            let at = self.cursor;
            self.buffer.insert(at, c);
            self.cursor += 1;
            self.modified = true;
            self.undo_stack.push(EditOp::Insert { at, text: c.to_string() });
            self.redo_stack.clear();
            return;
        }
        if c == ' ' {
            self.commit_pending();
        }
        let at = self.cursor;
        self.buffer.insert(at, c);
        self.cursor += 1;
        self.modified = true;
        self.redo_stack.clear();
        // Accumulate into pending burst
        match &mut self.pending_insert {
            Some((start, text)) if *start + text.chars().count() == at => {
                text.push(c);
            }
            _ => {
                self.commit_pending();
                self.pending_insert = Some((at, c.to_string()));
            }
        }
    }

    pub fn insert_str(&mut self, s: &str) {
        self.commit_pending();
        if self.selection().is_some() {
            let deleted = self.delete_selection().unwrap_or_default();
            if !deleted.is_empty() {
                self.undo_stack.push(EditOp::Delete { at: self.cursor, text: deleted });
            }
        }
        let at = self.cursor;
        for (i, c) in s.chars().enumerate() {
            self.buffer.insert(at + i, c);
        }
        self.cursor += s.chars().count();
        self.modified = true;
        self.undo_stack.push(EditOp::Insert { at, text: s.to_string() });
        self.redo_stack.clear();
    }

    pub fn backspace(&mut self) {
        self.commit_pending();
        if self.selection().is_some() {
            let deleted = self.delete_selection().unwrap_or_default();
            if !deleted.is_empty() {
                self.undo_stack.push(EditOp::Delete { at: self.cursor, text: deleted });
                self.redo_stack.clear();
            }
            return;
        }
        if self.cursor > 0 {
            let c = self.buffer.remove(self.cursor - 1);
            self.cursor -= 1;
            self.modified = true;
            self.undo_stack.push(EditOp::Delete { at: self.cursor, text: c.to_string() });
            self.redo_stack.clear();
        }
    }

    pub fn delete_forward(&mut self) {
        self.commit_pending();
        if self.selection().is_some() {
            let deleted = self.delete_selection().unwrap_or_default();
            if !deleted.is_empty() {
                self.undo_stack.push(EditOp::Delete { at: self.cursor, text: deleted });
                self.redo_stack.clear();
            }
            return;
        }
        if self.cursor < self.buffer.len() {
            let c = self.buffer.remove(self.cursor);
            self.modified = true;
            self.undo_stack.push(EditOp::Delete { at: self.cursor, text: c.to_string() });
            self.redo_stack.clear();
        }
    }

    // ─── Navigation ───────────────────────────────────────────────────────────

    pub fn move_left(&mut self, extend_selection: bool) {
        self.commit_pending();
        if extend_selection { self.start_selection(); } else { self.clear_selection(); }
        if self.cursor > 0 { self.cursor -= 1; }
    }

    pub fn move_right(&mut self, extend_selection: bool) {
        self.commit_pending();
        if extend_selection { self.start_selection(); } else { self.clear_selection(); }
        if self.cursor < self.buffer.len() { self.cursor += 1; }
    }

    pub fn move_up(&mut self, vlines: &[VisualLine], extend_selection: bool) {
        self.commit_pending();
        if extend_selection { self.start_selection(); } else { self.clear_selection(); }
        let cur = self.cursor_visual_line(vlines);
        if cur == 0 { return; }
        let col = self.cursor - vlines[cur].buf_start;
        let prev = &vlines[cur - 1];
        self.cursor = prev.buf_start + col.min(prev.buf_end - prev.buf_start);
    }

    pub fn move_down(&mut self, vlines: &[VisualLine], extend_selection: bool) {
        self.commit_pending();
        if extend_selection { self.start_selection(); } else { self.clear_selection(); }
        let cur = self.cursor_visual_line(vlines);
        if cur + 1 >= vlines.len() { return; }
        let col = self.cursor - vlines[cur].buf_start;
        let next = &vlines[cur + 1];
        self.cursor = next.buf_start + col.min(next.buf_end - next.buf_start);
    }

    pub fn home(&mut self, vlines: &[VisualLine], extend_selection: bool) {
        self.commit_pending();
        if extend_selection { self.start_selection(); } else { self.clear_selection(); }
        let cur = self.cursor_visual_line(vlines);
        self.cursor = vlines[cur].buf_start;
    }

    pub fn end(&mut self, vlines: &[VisualLine], extend_selection: bool) {
        self.commit_pending();
        if extend_selection { self.start_selection(); } else { self.clear_selection(); }
        let cur = self.cursor_visual_line(vlines);
        self.cursor = vlines[cur].buf_end;
    }

    pub fn page_up(&mut self, vlines: &[VisualLine], page_height: usize, extend_selection: bool) {
        self.commit_pending();
        if extend_selection { self.start_selection(); } else { self.clear_selection(); }
        let cur = self.cursor_visual_line(vlines);
        let target = cur.saturating_sub(page_height);
        let col = self.cursor - vlines[cur].buf_start;
        let vl = &vlines[target];
        self.cursor = vl.buf_start + col.min(vl.buf_end - vl.buf_start);
        self.scroll_offset = self.scroll_offset.saturating_sub(page_height);
    }

    pub fn page_down(&mut self, vlines: &[VisualLine], page_height: usize, extend_selection: bool) {
        self.commit_pending();
        if extend_selection { self.start_selection(); } else { self.clear_selection(); }
        let cur = self.cursor_visual_line(vlines);
        let target = (cur + page_height).min(vlines.len().saturating_sub(1));
        let col = self.cursor - vlines[cur].buf_start;
        let vl = &vlines[target];
        self.cursor = vl.buf_start + col.min(vl.buf_end - vl.buf_start);
    }

    pub fn select_all(&mut self) {
        self.commit_pending();
        self.anchor = Some(0);
        self.cursor = self.buffer.len();
    }

    pub fn word_left(&mut self, extend_selection: bool) {
        self.commit_pending();
        if extend_selection { self.start_selection(); } else { self.clear_selection(); }
        if self.cursor == 0 { return; }
        let mut pos = self.cursor;
        while pos > 0 && self.buffer[pos - 1].is_whitespace() { pos -= 1; }
        while pos > 0 && !self.buffer[pos - 1].is_whitespace() { pos -= 1; }
        self.cursor = pos;
    }

    pub fn word_right(&mut self, extend_selection: bool) {
        self.commit_pending();
        if extend_selection { self.start_selection(); } else { self.clear_selection(); }
        let len = self.buffer.len();
        if self.cursor >= len { return; }
        let mut pos = self.cursor;
        while pos < len && self.buffer[pos].is_whitespace() { pos += 1; }
        while pos < len && !self.buffer[pos].is_whitespace() { pos += 1; }
        self.cursor = pos;
    }

    // ─── Stats ────────────────────────────────────────────────────────────────

    pub fn word_count(&self) -> usize {
        let content: String = self.buffer.iter().collect();
        content.split_whitespace().count()
    }

    pub fn char_count(&self) -> usize {
        self.buffer.iter().filter(|&&c| c != '\n').count()
    }

    pub fn page_count(&self, text_height: usize, text_width: usize) -> usize {
        if text_height == 0 || text_width == 0 { return 1; }
        let total = self.visual_lines(text_width).len().max(1);
        (total + text_height - 1) / text_height
    }

    /// Returns (minutes, less_than_one). 200 wpm average.
    pub fn reading_time(&self) -> (usize, bool) {
        let words = self.word_count();
        if words == 0 { return (0, true); }
        ((words + 199) / 200, false)
    }

    // ─── File ─────────────────────────────────────────────────────────────────

    pub fn save(&mut self) -> io::Result<()> {
        if let Some(ref path) = self.file_path.clone() {
            let content: String = self.buffer.iter().collect();
            fs::write(path, content)?;
            self.modified = false;
        }
        Ok(())
    }

    pub fn save_as(&mut self, path: PathBuf) -> io::Result<()> {
        let content: String = self.buffer.iter().collect();
        fs::write(&path, &content)?;
        self.file_path = Some(path);
        self.modified  = false;
        Ok(())
    }

    // ─── Explorer ─────────────────────────────────────────────────────────────

    pub fn refresh_explorer(&mut self) {
        let mut entries: Vec<PathBuf> = fs::read_dir(&self.current_dir)
            .map(|rd| rd.filter_map(|e| e.ok().map(|e| e.path())).collect())
            .unwrap_or_default();

        entries.sort_by(|a, b| match (a.is_dir(), b.is_dir()) {
            (true, false) => std::cmp::Ordering::Less,
            (false, true) => std::cmp::Ordering::Greater,
            _ => a.file_name().cmp(&b.file_name()),
        });

        self.explorer_entries = entries;
        self.explorer_state   = ListState::default();
        if !self.explorer_entries.is_empty() {
            self.explorer_state.select(Some(0));
        }
    }

    pub fn explorer_selected(&self) -> Option<&PathBuf> {
        self.explorer_state.selected().and_then(|i| self.explorer_entries.get(i))
    }

    pub fn explorer_up(&mut self) {
        let i = self.explorer_state.selected().unwrap_or(0);
        if i > 0 { self.explorer_state.select(Some(i - 1)); }
    }

    pub fn explorer_down(&mut self) {
        let i = self.explorer_state.selected().unwrap_or(0);
        if i + 1 < self.explorer_entries.len() { self.explorer_state.select(Some(i + 1)); }
    }
}
