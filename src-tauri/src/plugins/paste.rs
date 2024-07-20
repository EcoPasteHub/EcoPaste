use rdev::{simulate, EventType, Key};
use std::{thread, time};
use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    Wry,
};

use super::window::focus_previous_window;

// 粘贴剪切板内容
#[command]
async fn paste() {
    focus_previous_window();

    if cfg!(target_os = "macos") {
        dispatch(&EventType::KeyPress(Key::MetaLeft));
        dispatch(&EventType::KeyPress(Key::KeyV));
        dispatch(&EventType::KeyRelease(Key::KeyV));
        dispatch(&EventType::KeyRelease(Key::MetaLeft));
    } else {
        dispatch(&EventType::KeyPress(Key::ControlLeft));
        dispatch(&EventType::KeyPress(Key::KeyV));
        dispatch(&EventType::KeyRelease(Key::KeyV));
        dispatch(&EventType::KeyRelease(Key::ControlLeft));
    }
}

fn dispatch(event_type: &EventType) {
    thread::sleep(time::Duration::from_millis(20));

    simulate(event_type).unwrap();
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("paste")
        .invoke_handler(generate_handler![paste])
        .build()
}
