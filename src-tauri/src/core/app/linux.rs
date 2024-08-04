use std::sync::Mutex;

use x11::xlib::{self, XDefaultRootWindow, XGetInputFocus, XNextEvent, XOpenDisplay, XSelectInput};

static FOREMOST_APPS: Mutex<Vec<u64>> = Mutex::new(Vec::new());

pub fn observe_app() {
    std::thread::spawn(|| unsafe {
        let display = XOpenDisplay(std::ptr::null_mut());
        if display.is_null() {
            eprintln!("Could not open display");
            return;
        }

        let root_window = XDefaultRootWindow(display);
        XSelectInput(display, root_window, xlib::FocusChangeMask);

        loop {
            let mut event = std::mem::zeroed();
            XNextEvent(display, &mut event);

            if event.get_type() != xlib::FocusIn {
                continue;
            }

            let mut app = FOREMOST_APPS.lock().unwrap();

            if app.len() >= 2 {
                app.remove(0);
            }

            let mut focus_return: u64 = 0;
            let mut revert_to_return: i32 = 0;
            XGetInputFocus(display, &mut focus_return, &mut revert_to_return);

            app.push(focus_return);
        }
    });
}

pub fn get_foreground_apps() -> Vec<u64> {
    return FOREMOST_APPS.lock().unwrap().to_vec();
}
