mod app;
mod types;
mod ui;

use std::{io, path::PathBuf};

use crossterm::{
    event::{self, Event, KeyCode, KeyModifiers},
    terminal::{disable_raw_mode, enable_raw_mode},
    ExecutableCommand,
};
use ratatui::{backend::CrosstermBackend, Terminal};

use app::App;
use types::Mode;

fn main() -> io::Result<()> {
    let args: Vec<String> = std::env::args().collect();
    let mut app = if args.len() > 1 {
        App::open_file(PathBuf::from(&args[1])).unwrap_or_else(|_| App::new())
    } else {
        App::new()
    };

    enable_raw_mode()?;
    io::stdout().execute(crossterm::terminal::EnterAlternateScreen)?;

    let backend = CrosstermBackend::new(io::stdout());
    let mut terminal = Terminal::new(backend)?;

    'main: loop {
        ui::set_terminal_title(&app);
        terminal.draw(|f| ui::draw(f, &mut app))?;

        if event::poll(std::time::Duration::from_millis(16))? {
            if let Event::Key(key) = event::read()? {
                let shift = key.modifiers.contains(KeyModifiers::SHIFT);
                let ctrl  = key.modifiers.contains(KeyModifiers::CONTROL);

                let approx_width = {
                    let (w, _) = crossterm::terminal::size().unwrap_or((80, 24));
                    let page_w = 90u16.min(w.saturating_sub(4));
                    page_w.saturating_sub(8) as usize
                };
                let approx_height = {
                    let (_, h) = crossterm::terminal::size().unwrap_or((80, 24));
                    h.saturating_sub(8) as usize
                };
                let vlines = app.visual_lines(approx_width);

                match app.mode {
                    Mode::Write => match key.code {
                        // ── File ────────────────────────────────────────────
                        KeyCode::Char('q') if ctrl => {
                            if app.modified { app.mode = Mode::ConfirmQuit; } else { break 'main; }
                        }
                        KeyCode::Char('s') if ctrl => {
                            if app.file_path.is_some() { let _ = app.save(); }
                            else {
                                app.input.clear();
                                app.save_as_input_focused = false;
                                app.refresh_explorer();
                                app.mode = Mode::SaveAs;
                            }
                        }
                        KeyCode::Char('o') if ctrl => {
                            app.refresh_explorer();
                            app.mode = Mode::Open;
                        }
                        KeyCode::Char('n') if ctrl => {
                            if app.modified { app.mode = Mode::ConfirmNew; }
                            else { app.new_document(); }
                        }

                        // ── Edit ────────────────────────────────────────────
                        KeyCode::Char('z') if ctrl => app.undo(),
                        KeyCode::Char('y') if ctrl => app.redo(),
                        KeyCode::Char('a') if ctrl => app.select_all(),
                        KeyCode::Char('c') if ctrl => app.copy(),
                        KeyCode::Char('x') if ctrl => app.cut(),
                        KeyCode::Char('v') if ctrl => app.paste(),

                        // ── Stats ───────────────────────────────────────────
                        KeyCode::Char('t') if ctrl => app.mode = Mode::Stats,

                        // ── Navigation ──────────────────────────────────────
                        KeyCode::Left if ctrl  => app.word_left(shift),
                        KeyCode::Right if ctrl => app.word_right(shift),
                        KeyCode::Left          => app.move_left(shift),
                        KeyCode::Right         => app.move_right(shift),
                        KeyCode::Up            => app.move_up(&vlines, shift),
                        KeyCode::Down          => app.move_down(&vlines, shift),
                        KeyCode::Home          => app.home(&vlines, shift),
                        KeyCode::End           => app.end(&vlines, shift),
                        KeyCode::PageUp        => app.page_up(&vlines, approx_height, shift),
                        KeyCode::PageDown      => app.page_down(&vlines, approx_height, shift),

                        // ── Input ───────────────────────────────────────────
                        KeyCode::Enter     => app.insert_char('\n'),
                        KeyCode::Backspace => app.backspace(),
                        KeyCode::Delete    => app.delete_forward(),
                        KeyCode::Tab       => app.insert_str("    "),
                        KeyCode::Char(c) if !ctrl => app.insert_char(c),
                        _ => {}
                    },

                    Mode::Open => match key.code {
                        KeyCode::Up    => app.explorer_up(),
                        KeyCode::Down  => app.explorer_down(),
                        KeyCode::Enter => {
                            if let Some(selected) = app.explorer_selected().cloned() {
                                if selected.is_dir() {
                                    app.current_dir = selected;
                                    app.refresh_explorer();
                                } else if let Ok(content) = std::fs::read_to_string(&selected) {
                                    app.buffer = content.chars().collect();
                                    app.cursor = app.buffer.len();
                                    app.file_path = Some(selected);
                                    app.modified  = false;
                                    app.scroll_offset = 0;
                                    app.pending_insert = None;
                                    app.undo_stack.clear();
                                    app.redo_stack.clear();
                                    app.mode = Mode::Write;
                                }
                            }
                        }
                        KeyCode::Esc => app.mode = Mode::Write,
                        _ => {}
                    },

                    Mode::SaveAs => {
                        if app.save_as_input_focused {
                            // Focus on filename input
                            match key.code {
                                KeyCode::Enter => {
                                    if !app.input.is_empty() {
                                        let path = app.current_dir.join(&app.input);
                                        let _ = app.save_as(path);
                                        app.mode = Mode::Write;
                                    }
                                }
                                KeyCode::Backspace => { app.input.pop(); }
                                KeyCode::Tab | KeyCode::BackTab => {
                                    app.save_as_input_focused = false;
                                }
                                KeyCode::Esc => app.mode = Mode::Write,
                                KeyCode::Char(c) => app.input.push(c),
                                _ => {}
                            }
                        } else {
                            // Focus on explorer
                            match key.code {
                                KeyCode::Up   => app.explorer_up(),
                                KeyCode::Down => app.explorer_down(),
                                KeyCode::Enter => {
                                    if let Some(selected) = app.explorer_selected().cloned() {
                                        if selected.is_dir() {
                                            app.current_dir = selected;
                                            app.refresh_explorer();
                                        } else {
                                            // Selected an existing file — use its name
                                            if let Some(name) = selected.file_name() {
                                                app.input = name.to_string_lossy().to_string();
                                            }
                                            app.save_as_input_focused = true;
                                        }
                                    }
                                }
                                KeyCode::Tab | KeyCode::BackTab => {
                                    app.save_as_input_focused = true;
                                }
                                KeyCode::Esc => app.mode = Mode::Write,
                                _ => {}
                            }
                        }
                    }

                    Mode::ConfirmQuit => match key.code {
                        KeyCode::Char('y') => break 'main,
                        KeyCode::Char('n') | KeyCode::Esc => app.mode = Mode::Write,
                        _ => {}
                    },

                    Mode::ConfirmNew => match key.code {
                        KeyCode::Char('y') => { app.new_document(); app.mode = Mode::Write; }
                        KeyCode::Char('n') | KeyCode::Esc => app.mode = Mode::Write,
                        _ => {}
                    },

                    Mode::Stats => match key.code {
                        KeyCode::Enter | KeyCode::Esc | KeyCode::Char('t') => {
                            app.mode = Mode::Write;
                        }
                        _ => {}
                    },
                }
            }
        }
    }

    disable_raw_mode()?;
    io::stdout().execute(crossterm::terminal::LeaveAlternateScreen)?;
    Ok(())
}
