use tauri::{App, Window};

pub fn platform(app: &mut App, main_window: Window, _preference_window: Window) {
    // 隐藏 mac 的程序坞图标：https://github.com/tauri-apps/tauri/issues/4852#issuecomment-1312716378
    app.set_activation_policy(tauri::ActivationPolicy::Accessory);

    unsafe {
        use cocoa::appkit::{NSMainMenuWindowLevel, NSWindow};
        use cocoa::base::id;

        let ns_window = main_window.ns_window().unwrap() as id;

        // 让窗口在程序坞和菜单栏之上
        ns_window.setLevel_(NSMainMenuWindowLevel as i64 + 1);
    }
}
