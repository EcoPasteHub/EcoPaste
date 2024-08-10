use std::sync::Mutex;

use x11::xlib::{
    self, Atom, Display, XDefaultRootWindow, XFree, XGetInputFocus, XGetWindowProperty,
    XInternAtom, XNextEvent, XOpenDisplay, XSelectInput,
};

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
            std::ffi::CStr::from_ptr(prop as *const i8)
                .to_string_lossy()
                .into_owned()
        };
        unsafe { XFree(prop as *mut _) };
        Ok(name)
    } else {
        Err(format!("{}", window))
    }
}

static FOREMOST_APPS: Mutex<Option<u64>> = Mutex::new(None);

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

            let mut focus_return: u64 = 0;
            let mut revert_to_return: i32 = 0;
            XGetInputFocus(display, &mut focus_return, &mut revert_to_return);

            if get_net_wm_name(display, focus_return).is_ok_and(|s| s.eq("EcoPaste")) {
                continue;
            }

            let mut app = FOREMOST_APPS.lock().unwrap();
            let _ = app.insert(focus_return);
        }
    });
}

pub fn get_foreground_apps() -> Option<u64> {
    return FOREMOST_APPS.lock().unwrap().clone();
}
