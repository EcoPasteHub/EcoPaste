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
            commands::save_window_state,
            commands::restore_window_state
        ])
        .on_window_ready(|window| {
            let app_handle = window.app_handle().clone();

            restore_window_state(window.clone());

            window.on_window_event(move |e| match e {
                // 普通的 tauri 窗口触发失去焦点事件
                WindowEvent::Focused(focused) => {
                    if *focused {
                        return;
                    }

                    save_window_state(app_handle.clone());
                }
                _ => {}
            });
        })
        .build()
}
