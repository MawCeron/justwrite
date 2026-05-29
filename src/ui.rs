use ratatui::{
    Frame,
    layout::{Alignment, Constraint, Direction, Layout, Rect},
    style::{Color, Style},
    text::{Line, Span},
    widgets::{Block, Borders, Clear, List, ListItem, Paragraph},
};

use crate::app::App;
use crate::types::{Mode, VisualLine};

// ─── Palette ──────────────────────────────────────────────────────────────────

const BG: Color            = Color::Black;
const PAGE_BG: Color       = Color::Black;
const BORDER: Color        = Color::Rgb(45, 45, 40);
const BORDER_MODIFIED: Color = Color::Rgb(90, 80, 55);
const PANEL_BG: Color      = Color::Rgb(22, 22, 20);
const PANEL_BORDER: Color  = Color::Rgb(90, 90, 80);
const TEXT: Color          = Color::Rgb(215, 215, 200);
const TEXT_DIM: Color      = Color::Rgb(110, 110, 100);
const CURSOR_BG: Color     = Color::Rgb(200, 200, 185);
const SELECT_BG: Color     = Color::Rgb(60, 70, 90);
const HIGHLIGHT_BG: Color  = Color::Rgb(55, 55, 50);
const DANGER_BORDER: Color = Color::Rgb(130, 70, 70);

// ─── Layout ───────────────────────────────────────────────────────────────────

pub fn centered_rect(percent_x: u16, percent_y: u16, r: Rect) -> Rect {
    let vert = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Percentage((100 - percent_y) / 2),
            Constraint::Percentage(percent_y),
            Constraint::Percentage((100 - percent_y) / 2),
        ])
        .split(r);

    Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Percentage((100 - percent_x) / 2),
            Constraint::Percentage(percent_x),
            Constraint::Percentage((100 - percent_x) / 2),
        ])
        .split(vert[1])[1]
}

/// Centered page: wide, subtle border, same background as terminal.
pub fn page_rect(r: Rect) -> Rect {
    let page_width = 90u16.min(r.width.saturating_sub(4));
    let x = r.width.saturating_sub(page_width) / 2;
    Rect { x, y: 2, width: page_width, height: r.height.saturating_sub(4) }
}

pub fn text_area(page: Rect) -> Rect {
    Rect {
        x: page.x + 4,
        y: page.y + 2,
        width:  page.width.saturating_sub(8),
        height: page.height.saturating_sub(4),
    }
}

// ─── Buffer render ────────────────────────────────────────────────────────────

pub fn render_visible_lines(
    app: &App,
    vlines: &[VisualLine],
    scroll: usize,
    height: usize,
) -> Vec<Line<'static>> {
    let sel = app.selection();
    let mut out = Vec::new();

    for vl in vlines.iter().skip(scroll).take(height) {
        let line_chars: Vec<char> = app.buffer[vl.buf_start..vl.buf_end].to_vec();
        let mut spans: Vec<Span<'static>> = Vec::new();
        let mut i = 0usize;

        while i <= line_chars.len() {
            let buf_pos = vl.buf_start + i;
            let is_cursor = buf_pos == app.cursor;
            let in_sel = sel.map(|(s, e)| buf_pos >= s && buf_pos < e).unwrap_or(false);

            let ch: String = if i < line_chars.len() {
                line_chars[i].to_string()
            } else if is_cursor {
                " ".to_string()
            } else {
                break;
            };

            let style = if is_cursor {
                Style::default().bg(CURSOR_BG).fg(Color::Black)
            } else if in_sel {
                Style::default().bg(SELECT_BG).fg(TEXT)
            } else {
                Style::default().fg(TEXT)
            };

            // Merge consecutive spans with same style to reduce allocations
            if let Some(last) = spans.last_mut() {
                if last.style == style {
                    let s = last.content.to_mut();
                    s.push_str(&ch);
                    i += 1;
                    continue;
                }
            }

            spans.push(Span::styled(ch, style));
            i += 1;
        }

        out.push(Line::from(spans));
    }

    out
}

// ─── Terminal title ───────────────────────────────────────────────────────────

pub fn set_terminal_title(app: &App) {
    let name = app.file_path
        .as_ref()
        .and_then(|p| p.file_name())
        .map(|n| n.to_string_lossy().to_string())
        .unwrap_or_else(|| "untitled".to_string());

    let title = if app.modified { format!("{}* — justwrite", name) } else { format!("{} — justwrite", name) };
    print!("\x1b]0;{}\x07", title);
}

// ─── Draw ─────────────────────────────────────────────────────────────────────

pub fn draw(f: &mut Frame, app: &mut App) {
    let size = f.size();

    f.render_widget(Block::default().style(Style::default().bg(BG)), size);

    // Page border changes color when modified
    let page = page_rect(size);
    let border_color = if app.modified { BORDER_MODIFIED } else { BORDER };
    f.render_widget(
        Block::default()
            .borders(Borders::ALL)
            .border_style(Style::default().fg(border_color))
            .style(Style::default().bg(PAGE_BG)),
        page,
    );

    let ta     = text_area(page);
    let width  = ta.width as usize;
    let height = ta.height as usize;

    let vlines = app.visual_lines(width);
    app.adjust_scroll(&vlines, height);
    let scroll = app.scroll_offset;

    let rendered = render_visible_lines(app, &vlines, scroll, height);

    f.render_widget(
        Paragraph::new(rendered)
            .alignment(Alignment::Left)
            .style(Style::default().fg(TEXT).bg(PAGE_BG)),
        ta,
    );

    match app.mode {
        Mode::Open        => draw_panel_open(f, app, size),
        Mode::SaveAs      => draw_panel_save_as(f, app, size),
        Mode::Stats       => draw_panel_stats(f, app, size, height, width),
        Mode::ConfirmQuit => draw_panel_confirm(f, size, "  quit without saving?  y / n"),
        Mode::ConfirmNew  => draw_panel_confirm(f, size, "  discard changes?  y / n"),
        Mode::Write       => {}
    }
}

// ─── Panels ───────────────────────────────────────────────────────────────────

fn draw_panel_open(f: &mut Frame, app: &mut App, size: Rect) {
    let panel = centered_rect(50, 60, size);
    f.render_widget(Clear, panel);

    let items: Vec<ListItem> = app.explorer_entries.iter().map(|p| {
        let name = p.file_name().map(|n| n.to_string_lossy().to_string()).unwrap_or_default();
        let label = if p.is_dir() { format!("  {}/", name) } else { format!("  {}", name) };
        ListItem::new(label)
    }).collect();

    let list = List::new(items)
        .block(Block::default()
            .borders(Borders::ALL)
            .border_style(Style::default().fg(PANEL_BORDER))
            .title(" open ")
            .title_alignment(Alignment::Center)
            .style(Style::default().bg(PANEL_BG)))
        .highlight_style(Style::default().bg(HIGHLIGHT_BG).fg(Color::White));

    f.render_stateful_widget(list, panel, &mut app.explorer_state);
}

fn draw_panel_save_as(f: &mut Frame, app: &App, size: Rect) {
    let panel = centered_rect(50, 70, size);
    f.render_widget(Clear, panel);

    // Split panel: explorer on top, filename input at bottom
    let chunks = ratatui::layout::Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Min(3),
            Constraint::Length(3),
        ])
        .split(panel);

    // ── Explorer ──────────────────────────────────────────────────────────────
    let items: Vec<ListItem> = app.explorer_entries.iter().map(|p| {
        let name = p.file_name().map(|n| n.to_string_lossy().to_string()).unwrap_or_default();
        let label = if p.is_dir() { format!("  {}/", name) } else { format!("  {}", name) };
        ListItem::new(label)
    }).collect();

    let explorer_border = if app.save_as_input_focused {
        Style::default().fg(PANEL_BORDER)
    } else {
        Style::default().fg(Color::Rgb(140, 130, 90)) // amber when focused
    };

    // Need mutable state for highlight — cast away const for render
    let mut state = app.explorer_state.clone();
    let list = List::new(items)
        .block(Block::default()
            .borders(Borders::ALL)
            .border_style(explorer_border)
            .title(" save as ")
            .title_alignment(Alignment::Center)
            .style(Style::default().bg(PANEL_BG)))
        .highlight_style(Style::default().bg(HIGHLIGHT_BG).fg(Color::White));

    f.render_stateful_widget(list, chunks[0], &mut state);

    // ── Filename input ────────────────────────────────────────────────────────
    let input_border = if app.save_as_input_focused {
        Style::default().fg(Color::Rgb(140, 130, 90)) // amber when focused
    } else {
        Style::default().fg(PANEL_BORDER)
    };

    let display = format!(" {}_", app.input);
    f.render_widget(
        Paragraph::new(display)
            .block(Block::default()
                .borders(Borders::ALL)
                .border_style(input_border)
                .title(" filename ")
                .title_alignment(Alignment::Center)
                .style(Style::default().bg(PANEL_BG)))
            .style(Style::default().fg(TEXT)),
        chunks[1],
    );
}

fn draw_panel_stats(f: &mut Frame, app: &App, size: Rect, height: usize, width: usize) {
    let panel = centered_rect(36, 32, size);
    f.render_widget(Clear, panel);

    let words = app.word_count();
    let chars = app.char_count();
    let pages = app.page_count(height, width);
    let (mins, short) = app.reading_time();
    let reading = if short { "< 1 min".to_string() } else { format!("~{} min", mins) };

    let dim = Style::default().fg(TEXT_DIM);
    let val = Style::default().fg(TEXT);

    let content = vec![
        Line::from(""),
        Line::from(vec![Span::styled("  words      ", dim), Span::styled(words.to_string(), val)]),
        Line::from(""),
        Line::from(vec![Span::styled("  characters ", dim), Span::styled(chars.to_string(), val)]),
        Line::from(""),
        Line::from(vec![Span::styled("  pages      ", dim), Span::styled(pages.to_string(), val)]),
        Line::from(""),
        Line::from(vec![Span::styled("  read time  ", dim), Span::styled(reading, val)]),
        Line::from(""),
    ];

    f.render_widget(
        Paragraph::new(content)
            .block(Block::default()
                .borders(Borders::ALL)
                .border_style(Style::default().fg(PANEL_BORDER))
                .title(" stats ")
                .title_alignment(Alignment::Center)
                .style(Style::default().bg(PANEL_BG)))
            .style(Style::default().bg(PANEL_BG)),
        panel,
    );
}

fn draw_panel_confirm(f: &mut Frame, size: Rect, message: &str) {
    let panel = centered_rect(40, 15, size);
    f.render_widget(Clear, panel);

    f.render_widget(
        Paragraph::new(message.to_string())
            .block(Block::default()
                .borders(Borders::ALL)
                .border_style(Style::default().fg(DANGER_BORDER))
                .style(Style::default().bg(PANEL_BG)))
            .style(Style::default().fg(TEXT)),
        panel,
    );
}
