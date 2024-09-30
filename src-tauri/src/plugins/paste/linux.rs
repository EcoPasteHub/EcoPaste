use super::wait;
use crate::core::app::get_previous_window;
use rdev::{simulate, EventType, Key};
use tauri::command;
use x11::xlib::{self, XCloseDisplay, XOpenDisplay, XRaiseWindow, XSetInputFocus};

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
