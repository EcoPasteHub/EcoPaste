use std::ptr::null_mut;
use tauri::{AppHandle, WebviewWindow};
use windows::Win32::Foundation::{HINSTANCE, LPARAM, LRESULT, WPARAM};
use windows::Win32::UI::WindowsAndMessaging::{
    CallNextHookEx, DispatchMessageW, GetMessageW, SetWindowsHookExW, TranslateMessage,
    KBDLLHOOKSTRUCT, MSG, WH_KEYBOARD_LL, WM_KEYDOWN,
};

unsafe extern "system" fn keyboard_proc(code: i32, w_param: WPARAM, l_param: LPARAM) -> LRESULT {
    if code >= 0 && w_param.0 == WM_KEYDOWN as usize {
        let kb_struct = *(l_param.0 as *const KBDLLHOOKSTRUCT);
        println!("Key pressed: {:?}", kb_struct.vkCode);
    }
    CallNextHookEx(None, code, w_param, l_param)
}

pub fn platform(
    _app_handle: &AppHandle,
    _main_window: WebviewWindow,
    _preference_window: WebviewWindow,
) {
    unsafe {
        let hook = SetWindowsHookExW(WH_KEYBOARD_LL, Some(keyboard_proc), HINSTANCE(0), 0);
        if hook.0 == 0 {
            eprintln!("Failed to install hook");
            return;
        }

        let mut msg = MSG::default();
        while GetMessageW(&mut msg, None, 0, 0).into() {
            TranslateMessage(&msg);
            DispatchMessageW(&msg);
        }
    }
}
