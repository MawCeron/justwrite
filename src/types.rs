// ─── Modes ────────────────────────────────────────────────────────────────────

pub enum Mode {
    Write,
    Open,
    SaveAs,
    ConfirmQuit,
    ConfirmNew,
    Stats,
}

// ─── Visual line ──────────────────────────────────────────────────────────────

#[derive(Clone)]
pub struct VisualLine {
    pub buf_start: usize,
    pub buf_end:   usize,
    pub hard_break: bool,
}

// ─── Undo/redo ────────────────────────────────────────────────────────────────

#[derive(Clone)]
pub enum EditOp {
    /// Inserted `text` at buffer position `at`.
    Insert { at: usize, text: String },
    /// Deleted `text` that was at buffer position `at`.
    Delete { at: usize, text: String },
}
