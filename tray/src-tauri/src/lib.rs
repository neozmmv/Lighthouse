use std::{
    io::{BufRead, BufReader},
    path::PathBuf,
    process::{Command, Stdio},
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
    mode: Mode,
    running: bool,
    starting: bool, // stdout reader thread is active; poller leaves state alone
    url: Option<String>,
}

fn truncate(s: &str) -> String {
    if s.len() > 50 {
        format!("{}...", &s[..47])
    } else {
        s.to_owned()
    }
}

// on disk daemon state

fn lighthouse_dir() -> Option<PathBuf> {
    Some(PathBuf::from(std::env::var("APPDATA").ok()?).join("lighthouse"))
}

fn read_pid() -> Option<u32> {
    std::fs::read_to_string(lighthouse_dir()?.join("lighthouse.pid"))
        .ok()?
        .trim()
        .parse()
        .ok()
}

fn is_pid_alive(pid: u32) -> bool {
    let mut cmd = Command::new("tasklist");
    cmd.args(["/FI", &format!("PID eq {pid}"), "/NH"])
        .stdout(Stdio::piped())
        .stderr(Stdio::null());
    #[cfg(target_os = "windows")]
    cmd.creation_flags(CREATE_NO_WINDOW);
    cmd.output()
        .map(|o| String::from_utf8_lossy(&o.stdout).contains(&pid.to_string()))
        .unwrap_or(false)
}

/// Returns `Some((mode, url))` when the daemon is fully up, `None` otherwise.
/// Checks the PID file first, then reads whichever URL file was written by the daemon.
fn external_state() -> Option<(Mode, String)> {
    let pid = read_pid()?;
    if !is_pid_alive(pid) {
        return None;
    }
    let dir = lighthouse_dir()?;
    for (file, mode) in [("tor_url", Mode::Tor), ("tunnel_url", Mode::Cloudflare)] {
        if let Ok(url) = std::fs::read_to_string(dir.join(file)) {
            let url = url.trim().to_string();
            if !url.is_empty() {
                return Some((mode, url));
            }
        }
    }
    None
}

// ui helpers

fn apply_running(
    url: &str,
    mode: &Mode,
    status: &MenuItem<Wry>,
    log: &MenuItem<Wry>,
    start: &MenuItem<Wry>,
    stop: &MenuItem<Wry>,
    open: &MenuItem<Wry>,
    tor_c: &CheckMenuItem<Wry>,
    cf_c: &CheckMenuItem<Wry>,
) {
    let _ = status.set_text("● Running");
    let _ = log.set_text(truncate(&format!("Lighthouse is running at: {url}")));
    let _ = log.set_enabled(true);
    let _ = start.set_enabled(false);
    let _ = stop.set_enabled(true);
    let _ = open.set_enabled(true);
    match mode {
        Mode::Tor => {
            let _ = tor_c.set_checked(true);
            let _ = cf_c.set_checked(false);
        }
        Mode::Cloudflare => {
            let _ = tor_c.set_checked(false);
            let _ = cf_c.set_checked(true);
        }
    }
}

fn apply_stopped(
    status: &MenuItem<Wry>,
    log: &MenuItem<Wry>,
    start: &MenuItem<Wry>,
    stop: &MenuItem<Wry>,
    open: &MenuItem<Wry>,
) {
    let _ = status.set_text("● Stopped");
    let _ = log.set_text("");
    let _ = log.set_enabled(false);
    let _ = start.set_enabled(true);
    let _ = stop.set_enabled(false);
    let _ = open.set_enabled(false);
}

// do_start / do_stop

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
        Mode::Tor => {
            cmd.arg("up");
        }
        Mode::Cloudflare => {
            cmd.args(["up", "--tunnel"]);
        }
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
    drop(child); // process keeps running; we only need the stdout pipe

    {
        let mut s = state.lock().unwrap();
        s.running = false;
        s.starting = true;
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
                let url = line.split_whitespace().last().unwrap_or(&line).to_string();
                {
                    let mut s = state.lock().unwrap();
                    s.running = true;
                    s.starting = false;
                    s.url = Some(url);
                }
                let _ = log.set_text(truncate(&line));
                let _ = log.set_enabled(true);
                let _ = status.set_text("● Running");
                let _ = open.set_enabled(true);
            } else if !became_running {
                let _ = log.set_text(truncate(&line));
            }
        }
        if !became_running {
            // startup failed before printing the running line
            let mut s = state.lock().unwrap();
            s.running = false;
            s.starting = false;
            s.url = None;
            drop(s);
            apply_stopped(&status, &log, &start, &stop, &open);
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
    s.running = false;
    s.starting = false;
    s.url = None;
    drop(s);

    apply_stopped(&status, &log, &start, &stop, &open);
}

// background poller

fn start_poller(
    state: Arc<Mutex<AppState>>,
    status: MenuItem<Wry>,
    log: MenuItem<Wry>,
    start: MenuItem<Wry>,
    stop: MenuItem<Wry>,
    open: MenuItem<Wry>,
    tor_c: CheckMenuItem<Wry>,
    cf_c: CheckMenuItem<Wry>,
) {
    std::thread::spawn(move || loop {
        std::thread::sleep(std::time::Duration::from_secs(2));

        let (tray_running, tray_starting) = {
            let s = state.lock().unwrap();
            (s.running, s.starting)
        };

        // while our own stdout reader is active, it owns state transitions
        if tray_starting {
            continue;
        }

        match (tray_running, external_state()) {
            (true, None) => {
                // stopped from terminal
                let mut s = state.lock().unwrap();
                s.running = false;
                s.url = None;
                drop(s);
                apply_stopped(&status, &log, &start, &stop, &open);
            }
            (false, Some((mode, url))) => {
                // started from terminal
                let mut s = state.lock().unwrap();
                s.running = true;
                s.url = Some(url.clone());
                s.mode = mode.clone();
                drop(s);
                apply_running(
                    &url, &mode, &status, &log, &start, &stop, &open, &tor_c, &cf_c,
                );
            }
            _ => {} // already in sync
        }
    });
}

// tray entry point

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .setup(|app| {
            let status_item = MenuItem::with_id(app, "status", "● Stopped", false, None::<&str>)?;
            let log_item = MenuItem::with_id(app, "log", "", false, None::<&str>)?;
            let sep1 = PredefinedMenuItem::separator(app)?;
            let tor_item =
                CheckMenuItem::with_id(app, "mode_tor", "Tor", true, true, None::<&str>)?;
            let cf_item = CheckMenuItem::with_id(
                app,
                "mode_cf",
                "Cloudflare Tunnel",
                true,
                false,
                None::<&str>,
            )?;
            let sep2 = PredefinedMenuItem::separator(app)?;
            let start_item = MenuItem::with_id(app, "start", "Start", true, None::<&str>)?;
            let stop_item = MenuItem::with_id(app, "stop", "Stop", false, None::<&str>)?;
            let sep3 = PredefinedMenuItem::separator(app)?;
            let open_item = MenuItem::with_id(app, "open", "Open Panel", false, None::<&str>)?;
            let sep4 = PredefinedMenuItem::separator(app)?;
            let quit_item = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;

            let menu = Menu::with_items(
                app,
                &[
                    &status_item,
                    &log_item,
                    &sep1,
                    &tor_item,
                    &cf_item,
                    &sep2,
                    &start_item,
                    &stop_item,
                    &sep3,
                    &open_item,
                    &sep4,
                    &quit_item,
                ],
            )?;

            let state = Arc::new(Mutex::new(AppState {
                mode: Mode::Tor,
                running: false,
                starting: false,
                url: None,
            }));

            // sync with whatever is already running before the tray opened
            if let Some((mode, url)) = external_state() {
                let mut s = state.lock().unwrap();
                s.running = true;
                s.url = Some(url.clone());
                s.mode = mode.clone();
                drop(s);
                apply_running(
                    &url,
                    &mode,
                    &status_item,
                    &log_item,
                    &start_item,
                    &stop_item,
                    &open_item,
                    &tor_item,
                    &cf_item,
                );
            }

            // poll for external start/stop every 2 s
            start_poller(
                state.clone(),
                status_item.clone(),
                log_item.clone(),
                start_item.clone(),
                stop_item.clone(),
                open_item.clone(),
                tor_item.clone(),
                cf_item.clone(),
            );

            let st = status_item.clone();
            let lg = log_item.clone();
            let s_start = start_item.clone();
            let s_stop = stop_item.clone();
            let s_open = open_item.clone();
            let tor_c = tor_item.clone();
            let cf_c = cf_item.clone();
            let state_ev = state.clone();

            TrayIconBuilder::new()
                .menu(&menu)
                .show_menu_on_left_click(true)
                .icon(include_image!("icons/Lighthouse.png"))
                .tooltip("Lighthouse")
                .on_menu_event(move |app, event| match event.id.as_ref() {
                    "start" => {
                        let mode = state_ev.lock().unwrap().mode.clone();
                        do_start(
                            mode,
                            state_ev.clone(),
                            st.clone(),
                            lg.clone(),
                            s_start.clone(),
                            s_stop.clone(),
                            s_open.clone(),
                        );
                    }
                    "stop" => {
                        do_stop(
                            state_ev.clone(),
                            st.clone(),
                            lg.clone(),
                            s_start.clone(),
                            s_stop.clone(),
                            s_open.clone(),
                        );
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
                                state_ev.clone(),
                                st.clone(),
                                lg.clone(),
                                s_start.clone(),
                                s_stop.clone(),
                                s_open.clone(),
                            );
                            std::thread::spawn(move || {
                                do_stop(
                                    state2.clone(),
                                    st2.clone(),
                                    lg2.clone(),
                                    start2.clone(),
                                    stop2.clone(),
                                    open2.clone(),
                                );
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
                                state_ev.clone(),
                                st.clone(),
                                lg.clone(),
                                s_start.clone(),
                                s_stop.clone(),
                                s_open.clone(),
                            );
                            std::thread::spawn(move || {
                                do_stop(
                                    state2.clone(),
                                    st2.clone(),
                                    lg2.clone(),
                                    start2.clone(),
                                    stop2.clone(),
                                    open2.clone(),
                                );
                                std::thread::sleep(std::time::Duration::from_secs(1));
                                do_start(Mode::Cloudflare, state2, st2, lg2, start2, stop2, open2);
                            });
                        }
                    }
                    "open" => {
                        let _ = app.opener().open_url("http://localhost:4405", None::<&str>);
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
