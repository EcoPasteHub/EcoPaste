use rdev::{grab, Event, EventType, Key};
use std::{
    sync::atomic::{AtomicBool, Ordering},
    thread::spawn,
};
use tauri::{AppHandle, Emitter, WebviewWindow};

static CTRL_PRESSED: AtomicBool = AtomicBool::new(false);
static SHIFT_PRESSED: AtomicBool = AtomicBool::new(false);

#[derive(Clone, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct DispatchEvent {
    code: &'static str,
    key_code: u32,
}

pub fn platform(
    app_handle: &AppHandle,
    main_window: WebviewWindow,
    _preference_window: WebviewWindow,
) {
    let app_handle = app_handle.clone();

    spawn(move || {
        let callback = move |event: Event| -> Option<Event> {
            match event.event_type {
                EventType::KeyPress(key) => {
                    match key {
                        Key::ControlLeft | Key::ControlRight => {
                            CTRL_PRESSED.store(true, Ordering::Relaxed)
                        }
                        Key::ShiftLeft | Key::ShiftRight => {
                            SHIFT_PRESSED.store(true, Ordering::Relaxed)
                        }
                        _ => {}
                    }

                    if let Ok(true) = main_window.is_visible() {
                        if handle_hotkey(&app_handle, key) {
                            return None;
                        }
                    }

                    Some(event)
                }

                EventType::KeyRelease(key) => {
                    match key {
                        Key::ControlLeft | Key::ControlRight => {
                            CTRL_PRESSED.store(false, Ordering::Relaxed)
                        }
                        Key::ShiftLeft | Key::ShiftRight => {
                            SHIFT_PRESSED.store(false, Ordering::Relaxed)
                        }
                        _ => {}
                    }
                    Some(event)
                }

                _ => Some(event),
            }
        };

        if let Err(err) = grab(callback) {
            eprintln!("rdev grab error: {:?}", err);
        }
    });
}

fn handle_hotkey(app_handle: &AppHandle, key: Key) -> bool {
    use Key::*;

    let ctrl_pressed = CTRL_PRESSED.load(Ordering::Relaxed);
    let shift_pressed = SHIFT_PRESSED.load(Ordering::Relaxed);

    let event = match key {
        // 预览
        Space => Some(DispatchEvent {
            code: "Space",
            key_code: 32,
        }),
        // 选择上一个
        UpArrow => Some(DispatchEvent {
            code: "ArrowUp",
            key_code: 38,
        }),
        // 选择下一个
        DownArrow => Some(DispatchEvent {
            code: "ArrowDown",
            key_code: 40,
        }),
        // 粘贴
        Return => Some(DispatchEvent {
            code: "Enter",
            key_code: 13,
        }),
        // 选择上一个分组
        Tab if shift_pressed => Some(DispatchEvent {
            code: "Tab",
            key_code: 9,
        }),
        // 选择下一个分组
        Tab => Some(DispatchEvent {
            code: "Tab",
            key_code: 9,
        }),
        // 滚动到顶部
        Home => Some(DispatchEvent {
            code: "Home",
            key_code: 36,
        }),
        // 删除条目
        Backspace => Some(DispatchEvent {
            code: "Backspace",
            key_code: 8,
        }),
        Delete => Some(DispatchEvent {
            code: "Delete",
            key_code: 46,
        }),
        // 收藏条目
        KeyD if ctrl_pressed => Some(DispatchEvent {
            code: "KeyD",
            key_code: 68,
        }),
        // 搜索框聚焦
        KeyF if ctrl_pressed => Some(DispatchEvent {
            code: "KeyF",
            key_code: 70,
        }),
        // 固定窗口
        KeyP if ctrl_pressed => Some(DispatchEvent {
            code: "KeyP",
            key_code: 80,
        }),
        // 打开偏好设置
        Comma if ctrl_pressed => Some(DispatchEvent {
            code: "Comma",
            key_code: 188,
        }),
        // 隐藏窗口
        KeyW if ctrl_pressed => Some(DispatchEvent {
            code: "KeyW",
            key_code: 87,
        }),
        _ => None,
    };

    if let Some(ev) = event {
        let _ = app_handle.emit("dispatch-event", ev);

        return true;
    }

    false
}
