use tauri::{
    image::Image,
    include_image,
    menu::{Menu, MenuItem, PredefinedMenuItem},
    tray::TrayIconBuilder,
};

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .setup(|app| {
            let open = MenuItem::with_id(app, "open", "Open Panel", true, None::<&str>)?;
            let up = MenuItem::with_id(app, "up", "Start Lighthouse", true, None::<&str>)?;
            let down = MenuItem::with_id(app, "down", "Stop Lighthouse", true, None::<&str>)?;
            let separator = PredefinedMenuItem::separator(app)?;
            let quit = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;

            let menu = Menu::with_items(app, &[&open, &up, &down, &separator, &quit])?;

            let icon = include_image!("icons/Lighthouse.png");

            TrayIconBuilder::new()
                .menu(&menu)
                .icon(icon)
                .tooltip("Lighthouse")
                .on_menu_event(|app, event| match event.id.as_ref() {
                    "open" => {
                        std::process::Command::new("cmd")
                            .args(["/c", "start", "http://localhost:4405"])
                            .spawn()
                            .unwrap();
                    }
                    "up" => {
                        std::process::Command::new("lighthouse")
                            .arg("up")
                            .spawn()
                            .unwrap();
                    }
                    "down" => {
                        std::process::Command::new("lighthouse")
                            .arg("down")
                            .spawn()
                            .unwrap();
                    }
                    "quit" => {
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
