use tauri::{ActivationPolicy, App, Emitter, Manager, WebviewWindow};
use tauri_nspanel::{
    cocoa::appkit::{NSMainMenuWindowLevel, NSWindowCollectionBehavior},
    panel_delegate, WebviewWindowExt,
};
use tauri_plugin_eco_window_state::save_window_state;

#[allow(non_upper_case_globals)]
const NSWindowStyleMaskNonActivatingPanel: i32 = 1 << 7;
#[allow(non_upper_case_globals)]
const NSResizableWindowMask: i32 = 1 << 3;
const MACOS_PANEL_FOCUS: &str = "macos-panel-focus";

pub fn platform(app: &mut App, main_window: WebviewWindow, _preference_window: WebviewWindow) {
    let app_handle = app.app_handle().clone();

    // 隐藏 mac 的程序坞图标：https://github.com/tauri-apps/tauri/issues/4852#issuecomment-1312716378
    app.set_activation_policy(ActivationPolicy::Accessory);

    // 把 ns_window 转换为 ns_panel
    let panel = main_window.to_panel().unwrap();

    // 让窗口在程序坞之上
    panel.set_level(NSMainMenuWindowLevel + 1);

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
        window_did_resign_key
    });

    // 为 delegate 设置事件监听器
    delegate.set_listener(Box::new(move |delegate_name: String| {
        match delegate_name.as_str() {
            // 当窗口获得键盘焦点时调用
            "window_did_become_key" => {
                app_handle.emit(MACOS_PANEL_FOCUS, true).unwrap();
            }
            // 当窗口失去键盘焦点时调用
            "window_did_resign_key" => {
                app_handle.emit(MACOS_PANEL_FOCUS, false).unwrap();

                let app_handle_clone = app_handle.clone();
                save_window_state(app_handle_clone);
            }
            _ => (),
        }
    }));

    // 设置窗口的委托对象，用于处理窗口的事件。
    panel.set_delegate(delegate);
}
