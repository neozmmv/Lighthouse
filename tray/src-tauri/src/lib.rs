use std::{
    io::{BufRead, BufReader},
    process::{Child, Command, Stdio},
    sync::{Arc, Mutex},
};
use tauri::{
    include_image,
    menu::{CheckMenuItem, Menu, MenuItem, PredefinedMenuItem},
    tray::TrayIconBuilder,
    Wry,
};
use tauri_plugin_opener::OpenerExt;

#[cfg(target_os = "windows")]
use std::os::windows::process::CommandExt;

#[cfg(target_os = "windows")]
const CREATE_NO_WINDOW: u32 = 0x08000000;

#[derive(Clone, PartialEq)]
enum Mode {
    Tor,
    Cloudflare,
}

struct AppState {
    child: Option<Child>,
    mode: Mode,
    running: bool,
    url: Option<String>,
}

fn truncate(s: &str) -> String {
    if s.len() > 50 {
        format!("{}...", &s[..47])
    } else {
        s.to_owned()
    }
}

fn do_start(
    mode: Mode,
    state: Arc<Mutex<AppState>>,
    status: MenuItem<Wry>,
    log: MenuItem<Wry>,
    start: MenuItem<Wry>,
    stop: MenuItem<Wry>,
    open: MenuItem<Wry>,
) {
    let mut cmd = Command::new("lighthouse");
    match mode {
        Mode::Tor => { cmd.arg("up"); }
        Mode::Cloudflare => { cmd.args(["up", "--tunnel"]); }
    }
    cmd.stdout(Stdio::piped());
    #[cfg(target_os = "windows")]
    cmd.creation_flags(CREATE_NO_WINDOW);

    let mut child = match cmd.spawn() {
        Ok(c) => c,
        Err(e) => {
            let _ = log.set_text(truncate(&format!("Error: {e}")));
            return;
        }
    };

    let stdout = child.stdout.take().expect("stdout should be piped");
    {
        let mut s = state.lock().unwrap();
        s.child = Some(child);
        s.running = false;
        s.url = None;
    }
    let _ = status.set_text("● Starting...");
    let _ = log.set_enabled(false);
    let _ = start.set_enabled(false);
    let _ = stop.set_enabled(true);

    std::thread::spawn(move || {
        let reader = BufReader::new(stdout);
        let mut became_running = false;
        for line in reader.lines() {
            let Ok(line) = line else { break };
            let line = line.trim().to_string();
            if line.is_empty() {
                continue;
            }
            if line.contains("Lighthouse is running at") {
                became_running = true;
                // extract bare URL (last whitespace-separated token)
                let url = line.split_whitespace().last().unwrap_or(&line).to_string();
                {
                    let mut s = state.lock().unwrap();
                    s.running = true;
                    s.url = Some(url);
                }
                let _ = log.set_text(truncate(&line)); // freeze here; ignore subsequent lines
                let _ = log.set_enabled(true); // clicking the label copies the URL
                let _ = status.set_text("● Running");
                let _ = open.set_enabled(true);
            } else if !became_running {
                let _ = log.set_text(truncate(&line)); // show progress during startup only
            }
        }
        // process exited — only reset if it never became running (startup failure)
        if !became_running {
            let _ = status.set_text("● Stopped");
            let _ = log.set_text("");
            let _ = log.set_enabled(false);
            let _ = start.set_enabled(true);
            let _ = stop.set_enabled(false);
            let _ = open.set_enabled(false);
            let mut s = state.lock().unwrap();
            s.child = None;
            s.running = false;
            s.url = None;
        }
    });
}

fn do_stop(
    state: Arc<Mutex<AppState>>,
    status: MenuItem<Wry>,
    log: MenuItem<Wry>,
    start: MenuItem<Wry>,
    stop: MenuItem<Wry>,
    open: MenuItem<Wry>,
) {
    let mut cmd = Command::new("lighthouse");
    cmd.arg("down");
    #[cfg(target_os = "windows")]
    cmd.creation_flags(CREATE_NO_WINDOW);
    let _ = cmd.spawn();

    let mut s = state.lock().unwrap();
    s.child = None;
    s.running = false;
    s.url = None;
    drop(s);

    let _ = status.set_text("● Stopped");
    let _ = log.set_text("");
    let _ = log.set_enabled(false);
    let _ = start.set_enabled(true);
    let _ = stop.set_enabled(false);
    let _ = open.set_enabled(false);
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .setup(|app| {
            let status_item = MenuItem::with_id(app, "status", "● Stopped", false, None::<&str>)?;
            // disabled until running; enabled then to copy URL on click
            let log_item    = MenuItem::with_id(app, "log",    "",           false, None::<&str>)?;
            let sep1        = PredefinedMenuItem::separator(app)?;
            let tor_item    = CheckMenuItem::with_id(app, "mode_tor", "Tor",               true, true,  None::<&str>)?;
            let cf_item     = CheckMenuItem::with_id(app, "mode_cf",  "Cloudflare Tunnel", true, false, None::<&str>)?;
            let sep2        = PredefinedMenuItem::separator(app)?;
            let start_item  = MenuItem::with_id(app, "start",  "Start",      true,  None::<&str>)?;
            let stop_item   = MenuItem::with_id(app, "stop",   "Stop",       false, None::<&str>)?;
            let sep3        = PredefinedMenuItem::separator(app)?;
            let open_item   = MenuItem::with_id(app, "open",   "Open Panel", false, None::<&str>)?;
            let sep4        = PredefinedMenuItem::separator(app)?;
            let quit_item   = MenuItem::with_id(app, "quit",   "Quit",       true,  None::<&str>)?;

            let menu = Menu::with_items(app, &[
                &status_item, &log_item, &sep1,
                &tor_item, &cf_item, &sep2,
                &start_item, &stop_item, &sep3,
                &open_item, &sep4,
                &quit_item,
            ])?;

            let state = Arc::new(Mutex::new(AppState {
                child: None,
                mode: Mode::Tor,
                running: false,
                url: None,
            }));

            let st       = status_item.clone();
            let lg       = log_item.clone();
            let s_start  = start_item.clone();
            let s_stop   = stop_item.clone();
            let s_open   = open_item.clone();
            let tor_c    = tor_item.clone();
            let cf_c     = cf_item.clone();
            let state_ev = state.clone();

            TrayIconBuilder::new()
                .menu(&menu)
                .show_menu_on_left_click(true)
                .icon(include_image!("icons/Lighthouse.png"))
                .tooltip("Lighthouse")
                .on_menu_event(move |app, event| match event.id.as_ref() {
                    "start" => {
                        let mode = state_ev.lock().unwrap().mode.clone();
                        do_start(mode, state_ev.clone(), st.clone(), lg.clone(), s_start.clone(), s_stop.clone(), s_open.clone());
                    }
                    "stop" => {
                        do_stop(state_ev.clone(), st.clone(), lg.clone(), s_start.clone(), s_stop.clone(), s_open.clone());
                    }
                    "log" => {
                        if let Some(url) = state_ev.lock().unwrap().url.clone() {
                            if let Ok(mut cb) = arboard::Clipboard::new() {
                                let _ = cb.set_text(url);
                            }
                        }
                    }
                    "mode_tor" => {
                        let needs_restart = {
                            let mut s = state_ev.lock().unwrap();
                            let restart = s.running && s.mode != Mode::Tor;
                            s.mode = Mode::Tor;
                            restart
                        };
                        let _ = tor_c.set_checked(true);
                        let _ = cf_c.set_checked(false);
                        if needs_restart {
                            let (state2, st2, lg2, start2, stop2, open2) = (
                                state_ev.clone(), st.clone(), lg.clone(), s_start.clone(), s_stop.clone(), s_open.clone(),
                            );
                            std::thread::spawn(move || {
                                do_stop(state2.clone(), st2.clone(), lg2.clone(), start2.clone(), stop2.clone(), open2.clone());
                                std::thread::sleep(std::time::Duration::from_secs(1));
                                do_start(Mode::Tor, state2, st2, lg2, start2, stop2, open2);
                            });
                        }
                    }
                    "mode_cf" => {
                        let needs_restart = {
                            let mut s = state_ev.lock().unwrap();
                            let restart = s.running && s.mode != Mode::Cloudflare;
                            s.mode = Mode::Cloudflare;
                            restart
                        };
                        let _ = tor_c.set_checked(false);
                        let _ = cf_c.set_checked(true);
                        if needs_restart {
                            let (state2, st2, lg2, start2, stop2, open2) = (
                                state_ev.clone(), st.clone(), lg.clone(), s_start.clone(), s_stop.clone(), s_open.clone(),
                            );
                            std::thread::spawn(move || {
                                do_stop(state2.clone(), st2.clone(), lg2.clone(), start2.clone(), stop2.clone(), open2.clone());
                                std::thread::sleep(std::time::Duration::from_secs(1));
                                do_start(Mode::Cloudflare, state2, st2, lg2, start2, stop2, open2);
                            });
                        }
                    }
                    "open" => {
                        let _ = app.opener().open_url("http://localhost:8080", None::<&str>);
                    }
                    "quit" => {
                        if state_ev.lock().unwrap().running {
                            let mut cmd = Command::new("lighthouse");
                            cmd.arg("down");
                            #[cfg(target_os = "windows")]
                            cmd.creation_flags(CREATE_NO_WINDOW);
                            let _ = cmd.spawn();
                        }
                        app.exit(0);
                    }
                    _ => {}
                })
                .build(app)?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
