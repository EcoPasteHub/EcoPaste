use super::wait;
use rdev::{simulate, EventType, Key};
use std::sync::Mutex;
use tauri::command;
use tauri_plugin_eco_window::MAIN_WINDOW_TITLE;
use x11::xlib::{
    self, Atom, Display, XCloseDisplay, XDefaultRootWindow, XFree, XGetInputFocus,
    XGetWindowProperty, XInternAtom, XNextEvent, XOpenDisplay, XRaiseWindow, XSelectInput,
    XSetInputFocus,
};

static PREVIOUS_WINDOW: Mutex<Option<u64>> = Mutex::new(None);

// 获取窗口标题
fn get_net_wm_name(display: *mut Display, window: u64) -> std::result::Result<String, String> {
    let mut actual_type: Atom = 0;
    let mut actual_format: i32 = 0;
    let mut nitems: u64 = 0;
    let mut bytes_after: u64 = 0;
    let mut prop: *mut u8 = std::ptr::null_mut();
    let net_wm_name_atom =
        unsafe { XInternAtom(display, b"_NET_WM_NAME\0".as_ptr() as _, xlib::False) };
    let result = unsafe {
        XGetWindowProperty(
            display,
            window,
            net_wm_name_atom,
            0,
            1024,
            xlib::False,
            xlib::AnyPropertyType as _,
            &mut actual_type,
            &mut actual_format,
            &mut nitems,
            &mut bytes_after,
            &mut prop,
        )
    };
    if result == xlib::Success as i32 && !prop.is_null() {
        let name = unsafe {
            std::ffi::CStr::from_ptr(prop as *const std::ffi::c_char)
                .to_string_lossy()
                .into_owned()
        };
        unsafe { XFree(prop as *mut _) };
        Ok(name)
    } else {
        Err(format!("{}", window))
    }
}

// 监听窗口切换
pub fn observe_app() {
    std::thread::spawn(|| unsafe {
        let display = XOpenDisplay(std::ptr::null_mut());
        if display.is_null() {
            log::error!("Could not open display");
            return;
        }

        let root_window = XDefaultRootWindow(display);
        XSelectInput(
            display,
            root_window,
            xlib::FocusChangeMask | xlib::PropertyChangeMask,
        );

        loop {
            let mut event = std::mem::zeroed();
            XNextEvent(display, &mut event);

            let mut window: u64 = 0;
            let mut revert_to_return: i32 = 0;
            XGetInputFocus(display, &mut window, &mut revert_to_return);

            if window == 1 {
                continue;
            }

            let wm_name = get_net_wm_name(display, window).unwrap_or_default();

            if wm_name.is_empty() || wm_name.eq(MAIN_WINDOW_TITLE) {
                continue;
            }

            let mut previous_window = PREVIOUS_WINDOW.lock().unwrap();
            let _ = previous_window.insert(window);
        }
    });
}

// 获取上一个窗口
pub fn get_previous_window() -> Option<u64> {
    return PREVIOUS_WINDOW.lock().unwrap().clone();
}

// 聚焦上一个窗口
fn focus_previous_window() {
    unsafe {
        let display = XOpenDisplay(std::ptr::null_mut());
        if display.is_null() {
            log::error!("Could not open display");
            return;
        }
        let window = match get_previous_window() {
            Some(window) => window,
            None => {
                log::error!("Could not get active window");
                return;
            }
        };

        XRaiseWindow(display, window);
        XSetInputFocus(display, window, xlib::RevertToNone, xlib::CurrentTime);
        XCloseDisplay(display);
    }
}

// 粘贴
#[command]
pub async fn paste() {
    fn dispatch(event_type: &EventType) {
        wait(20);

        simulate(event_type).unwrap();
    }

    focus_previous_window();

    wait(100);

    dispatch(&EventType::KeyPress(Key::ShiftLeft));
    dispatch(&EventType::KeyPress(Key::Insert));
    dispatch(&EventType::KeyRelease(Key::Insert));
    dispatch(&EventType::KeyRelease(Key::ShiftLeft));
}
