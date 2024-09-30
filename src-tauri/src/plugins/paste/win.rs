use super::wait;
use crate::core::app::get_previous_window;
use enigo::{
    Direction::{Click, Press, Release},
    Enigo, Key, Keyboard, Settings,
};
use tauri::command;
use winapi::{shared::windef::HWND, um::winuser::SetForegroundWindow};

fn focus_previous_window() {
    unsafe {
        let hwnd = match get_previous_window() {
            Some(hwnd) => hwnd as HWND,
            None => return,
        };

        if hwnd.is_null() {
            return;
        }

        SetForegroundWindow(hwnd);
    }
}

#[command]
pub async fn paste() {
    let mut enigo = Enigo::new(&Settings::default()).unwrap();

    focus_previous_window();

    wait(100);

    enigo.key(Key::Shift, Press).unwrap();
    // insert 的微软虚拟键码：https://learn.microsoft.com/en-us/windows/win32/inputdev/virtual-key-codes
    enigo.key(Key::Other(0x2D), Click).unwrap();
    enigo.key(Key::Shift, Release).unwrap();
}
