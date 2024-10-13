use super::wait;
use rdev::{simulate, EventType, Key};
use std::sync::Mutex;
use std::time::Instant;
use tauri::command;
use tauri_plugin_eco_window::MAIN_WINDOW_TITLE;
use x11::xlib::{
    self, Atom, Display, XCloseDisplay, XDefaultRootWindow, XFlush, XFree, XGetInputFocus,
    XGetWindowProperty, XInternAtom, XNextEvent, XOpenDisplay, XRaiseWindow, XSelectInput,
    XSetInputFocus, XSync,
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

            if wm_name.eq(MAIN_WINDOW_TITLE) {
                log::trace!("Ignoring window -> {}(0x{:x})", wm_name, window);
                continue;
            }

            if wm_name.is_empty() {
                log::trace!("Ignoring empty window: 0x{:x} .", window);
                continue;
            }

            match PREVIOUS_WINDOW.lock() {
                Ok(mut previous_window) => {
                    // 判断是否是同一个窗口
                    if previous_window.is_none() || previous_window.unwrap() != window {
                        let _ = previous_window.insert(window);
                        log::info!("Set previous window -> {}(0x{:x})", wm_name, window);
                    } else {
                        log::trace!("Ignoring same window -> {}(0x{:x})", wm_name, window);
                    }
                    drop(previous_window);
                }
                Err(e) => {
                    log::error!("Could not lock previous window: {}", e);
                }
            }
        }
    });
}

/**
 * 使用bash命令实现窗口聚焦
 */
#[allow(dead_code)]
#[allow(unused_variables)]
fn focus_window_with_command(window_name: &str) -> bool {
    let output = std::process::Command::new("/bin/wmctrl")
        .args(["-a", window_name])
        .output()
        .expect("Failed to set focus");

    return output.status.success();
}

/**
 * 使用原生x11实现窗口聚焦，不稳定
 */
#[allow(dead_code)]
#[allow(unused_variables)]
fn focus_window_with_x11(display: *mut Display, window: u64) -> bool {
    unsafe {
        XRaiseWindow(display, window);
        XSetInputFocus(display, window, xlib::RevertToNone, xlib::CurrentTime);
        // XSetInputFocus(display, window, xlib::RevertToParent, xlib::CurrentTime);

        // 刷新并同步 X11 事件，确保焦点切换及时生效
        XFlush(display);
        XSync(display, xlib::False);
    }
    return true;
}

/**
 * 获取窗口焦点
 */
#[allow(unused_variables)]
fn focus_window(display: *mut Display, window: u64, window_name: &str) -> bool {
    return focus_window_with_command(window_name);
    // return focus_window_with_x11(display, window);
}

/**
 * 获取窗口焦点，判断当前是否聚焦
 */
fn is_window_focused(display: *mut Display, window: u64) -> bool {
    unsafe {
        if display.is_null() {
            log::error!("Could not open display");
            return false;
        }

        let mut focused_window: u64 = 0;
        let mut revert_to_return: i32 = 0;
        XGetInputFocus(display, &mut focused_window, &mut revert_to_return);

        return focused_window == window;
    }
}

/**
 * 批量发送按键事件
 */
fn send(event_types: &[EventType]) {
    for event_type in event_types {
        match simulate(event_type) {
            Ok(()) => {
                log::info!("Send {:?} success!", event_type);
            }
            Err(e) => {
                log::error!("Send {:?} error {:?}!", event_type, e);
                return;
            }
        }
        // 短暂的延迟
        wait(20);
    }
}

// 粘贴
#[command]
pub async fn paste() {
    unsafe {
        let display: *mut xlib::_XDisplay = XOpenDisplay(std::ptr::null_mut());
        if display.is_null() {
            log::error!("Could not open display");
            return;
        }

        let window = match PREVIOUS_WINDOW.lock().ok().and_then(|w| *w) {
            Some(window) => window,
            None => {
                log::error!("Could not lock previous window");
                return;
            }
        };

        log::info!("Focusing on window: 0x{:x}", window);

        let window_name = get_net_wm_name(display, window).unwrap_or_default();

        let focused = focus_window(display, window, &window_name);

        if !focused {
            log::error!("Focus window {} error !!!!", &window_name);
            return;
        }

        let time = Instant::now();
        loop {
            // XRaiseWindow(display, window);
            XFlush(display);
            XSync(display, xlib::False);

            if time.elapsed().as_secs() >= 5 {
                log::error!(
                    "Focus window {} timeout {:?} !!!!",
                    window_name,
                    time.elapsed()
                );
                return;
            }

            if time.elapsed().as_millis() < 500 {
                log::trace!(
                    "Already cost {} ， Check {:?} focus state ...",
                    window_name,
                    time.elapsed()
                );
            } else {
                log::debug!(
                    "Window {} focus slow [{:?}], Checking ...",
                    window_name,
                    time.elapsed()
                );
            }

            if is_window_focused(display, window) {
                XRaiseWindow(display, window);
                log::info!(
                    "Window {} is successfully focused, using {:?}",
                    window_name,
                    time.elapsed()
                );
                break;
            }

            wait(10);
        }

        send(&[
            // 大写锁定测试
            // EventType::KeyPress(Key::CapsLock),
            // EventType::KeyRelease(Key::CapsLock)

            // 测试Ctrl + 1 组合键
            // EventType::KeyPress(Key::ControlLeft),
            // EventType::KeyPress(Key::Num1),
            // EventType::KeyRelease(Key::Num1),
            // EventType::KeyRelease(Key::ControlLeft),
            EventType::KeyPress(Key::ShiftLeft),
            EventType::KeyPress(Key::Insert),
            EventType::KeyRelease(Key::Insert),
            EventType::KeyRelease(Key::ShiftLeft),
            // EventType::KeyPress(Key::ControlLeft),
            // EventType::KeyPress(Key::KeyV),
            // EventType::KeyRelease(Key::KeyV),
            // EventType::KeyRelease(Key::ControlLeft),
        ]);

        if !is_window_focused(display, window) {
            log::warn!("Lose Window {} focus !!!", window_name);
        }

        XCloseDisplay(display);

        log::info!("Paste finished .");
    }
}
