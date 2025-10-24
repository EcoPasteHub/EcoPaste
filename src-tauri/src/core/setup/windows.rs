use std::ptr::null_mut;
use tauri::{AppHandle, WebviewWindow};
use windows::Win32::Foundation::{HINSTANCE, LPARAM, LRESULT, WPARAM};
use windows::Win32::UI::Input::KeyboardAndMouse::GetAsyncKeyState;
use windows::Win32::UI::WindowsAndMessaging::{
    CallNextHookEx, DispatchMessageW, GetMessageW, SetWindowsHookExW, TranslateMessage,
    KBDLLHOOKSTRUCT, MSG, WH_KEYBOARD_LL, WM_KEYDOWN, WM_KEYUP,
};

unsafe extern "system" fn keyboard_proc(code: i32, w_param: WPARAM, l_param: LPARAM) -> LRESULT {
    if code >= 0 {
        let kb_struct = *(l_param.0 as *const KBDLLHOOKSTRUCT);

        match w_param.0 as u32 {
            WM_KEYDOWN => {
                println!("Key pressed: {:?}", kb_struct.vkCode);
            }
            WM_KEYUP => {
                println!("Key released: {:?}", kb_struct.vkCode);
            }
            _ => {}
        }

        // 返回非零值阻止系统接收按键
        return LRESULT(1);
    }

    CallNextHookEx(None, code, w_param, l_param)
}

pub fn platform(
    _app_handle: &AppHandle,
    _main_window: WebviewWindow,
    _preference_window: WebviewWindow,
) {
    std::thread::spawn(|| unsafe {
        let hook = SetWindowsHookExW(
            WH_KEYBOARD_LL,
            Some(keyboard_proc),
            Some(HINSTANCE(null_mut())),
            0,
        );

        if let Err(err) = hook {
            eprintln!("Failed to install keyboard hook: {:?}", err);
            return;
        }

        let mut msg = MSG::default();
        while GetMessageW(&mut msg, None, 0, 0).into() {
            TranslateMessage(&msg);
            DispatchMessageW(&msg);
        }
    });
}
