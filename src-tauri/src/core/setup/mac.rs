use tauri::{ActivationPolicy, App, Emitter, EventTarget, Manager, WebviewWindow};
use tauri_nspanel::{cocoa::appkit::NSWindowCollectionBehavior, panel_delegate, WebviewWindowExt};
use tauri_plugin_eco_window::MAIN_WINDOW_LABEL;

#[allow(non_upper_case_globals)]
const NSWindowStyleMaskNonActivatingPanel: i32 = 1 << 7;
#[allow(non_upper_case_globals)]
const NSResizableWindowMask: i32 = 1 << 3;
const WINDOW_FOCUS_EVENT: &str = "tauri://focus";
const WINDOW_BLUR_EVENT: &str = "tauri://blur";
const WINDOW_MOVED_EVENT: &str = "tauri://move";
const WINDOW_RESIZED_EVENT: &str = "tauri://resize";

pub fn platform(app: &mut App, main_window: WebviewWindow, _preference_window: WebviewWindow) {
    let app_handle = app.app_handle().clone();

    // macos window 转 ns_panel 插件
    let _ = app_handle.plugin(tauri_nspanel::init());

    // 隐藏 mac 的程序坞图标：https://github.com/tauri-apps/tauri/issues/4852#issuecomment-1312716378
    app.set_activation_policy(ActivationPolicy::Accessory);

    // 把 ns_window 转换为 ns_panel
    let panel = main_window.to_panel().unwrap();

    // 让窗口在程序坞之上
    panel.set_level(20);

    // 不抢占其它窗口的焦点和支持缩放
    panel.set_style_mask(NSWindowStyleMaskNonActivatingPanel | NSResizableWindowMask);

    // 在各个桌面空间、全屏中共享窗口
    panel.set_collection_behaviour(
        NSWindowCollectionBehavior::NSWindowCollectionBehaviorCanJoinAllSpaces
            | NSWindowCollectionBehavior::NSWindowCollectionBehaviorStationary
            | NSWindowCollectionBehavior::NSWindowCollectionBehaviorFullScreenAuxiliary,
    );

    // 定义面板的委托 (delegate)，用于监听面板窗口的事件
    let delegate = panel_delegate!(EcoPanelDelegate {
        window_did_become_key,
        window_did_resign_key,
        window_did_resize,
        window_did_move
    });

    // 为 delegate 设置事件监听器
    delegate.set_listener(Box::new(move |delegate_name: String| {
        let target = EventTarget::labeled(MAIN_WINDOW_LABEL);

        let window_move_event = || {
            if let Ok(position) = main_window.outer_position() {
                let _ = main_window.emit_to(target.clone(), WINDOW_MOVED_EVENT, position);
            }
        };

        match delegate_name.as_str() {
            // 当窗口获得键盘焦点时调用
            "window_did_become_key" => {
                let _ = main_window.emit_to(target, WINDOW_FOCUS_EVENT, true);
            }
            // 当窗口失去键盘焦点时调用
            "window_did_resign_key" => {
                let _ = main_window.emit_to(target, WINDOW_BLUR_EVENT, true);
            }
            // 当窗口大小改变时调用
            "window_did_resize" => {
                window_move_event();

                if let Ok(size) = main_window.inner_size() {
                    let _ = main_window.emit_to(target, WINDOW_RESIZED_EVENT, size);
                }
            }
            // 当窗口位置改变时调用
            "window_did_move" => window_move_event(),
            _ => (),
        }
    }));

    // 设置窗口的委托对象，用于处理窗口的事件。
    panel.set_delegate(delegate);
}
