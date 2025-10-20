use rdev::{listen, Event, EventType, Key};
use std::thread;
use tauri::{AppHandle, Emitter, WebviewWindow};

pub fn platform(
    app_handle: &AppHandle,
    _main_window: WebviewWindow,
    _preference_window: WebviewWindow,
) {
    let app_handle = app_handle.clone();

    let mut win_pressed = false;

    thread::spawn(move || {
        let callback = move |event: Event| match event.event_type {
            EventType::KeyPress(key) => match key {
                Key::MetaLeft | Key::MetaRight => {
                    win_pressed = true;

                    println!("Win key pressed");
                }
                Key::KeyV => {
                    println!("V key pressed");

                    if win_pressed {
                        println!("Win + V pressed!");

                        let _ = app_handle.emit("toggle-main-window-visible", ());
                    }
                }
                _ => {}
            },
            EventType::KeyRelease(key) => match key {
                Key::MetaLeft | Key::MetaRight => {
                    win_pressed = false;

                    println!("Win key released");
                }
                _ => {}
            },
            _ => {}
        };

        if let Err(error) = listen(callback) {
            log::error!("{:?}", error);
        }
    });
}
