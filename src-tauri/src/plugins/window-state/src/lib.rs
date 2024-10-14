use tauri::{
    generate_handler,
    plugin::{Builder, TauriPlugin},
    Manager, Runtime, WindowEvent,
};

mod commands;

pub use commands::*;

pub fn init<R: Runtime>() -> TauriPlugin<R> {
    Builder::new("eco-window-state")
        .invoke_handler(generate_handler![
            commands::save_state,
            commands::restore_state
        ])
        .on_window_ready(|window| {
            let app_handle = window.app_handle().clone();

            if let Ok(app_data_dir) = app_handle.path().app_data_dir() {
                let save_path = app_data_dir.join(".window-state.json");

                restore_state(window.clone(), save_path.clone());

                window.on_window_event(move |e| match e {
                    WindowEvent::Focused(focused) => {
                        if *focused {
                            return;
                        }

                        save_state(app_handle.clone(), save_path.clone());
                    }
                    _ => {}
                });
            }
        })
        .build()
}
