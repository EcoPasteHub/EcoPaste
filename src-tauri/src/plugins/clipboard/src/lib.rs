use commands::ClipboardManager;
use tauri::{
    generate_handler,
    plugin::{Builder, TauriPlugin},
    Manager, Runtime,
};

mod commands;

pub fn init<R: Runtime>() -> TauriPlugin<R> {
    Builder::new("eco-clipboard")
        .setup(move |app, _api| {
            app.manage(ClipboardManager::new());

            Ok(())
        })
        .invoke_handler(generate_handler![
            commands::start_listen,
            commands::stop_listen,
            commands::has_files,
            commands::has_image,
            commands::has_html,
            commands::has_rtf,
            commands::has_text,
            commands::read_files,
            commands::read_image,
            commands::read_html,
            commands::read_rtf,
            commands::read_text,
            commands::write_files,
            commands::write_image,
            commands::write_html,
            commands::write_rtf,
            commands::write_text
        ])
        .build()
}
