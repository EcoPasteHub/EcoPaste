use cocoa::appkit::{NSMainMenuWindowLevel, NSWindow};
use cocoa::base::id;
use tauri::{ActivationPolicy, App, WebviewWindow};

pub fn platform(app: &mut App, main_window: WebviewWindow, _preference_window: WebviewWindow) {
    // 隐藏 mac 的程序坞图标：https://github.com/tauri-apps/tauri/issues/4852#issuecomment-1312716378
    app.set_activation_policy(ActivationPolicy::Accessory);

    unsafe {
        let ns_window = main_window.ns_window().unwrap() as id;

        // 让窗口在程序坞和菜单栏之上
        ns_window.setLevel_(NSMainMenuWindowLevel as i64 + 1);
    }
}
