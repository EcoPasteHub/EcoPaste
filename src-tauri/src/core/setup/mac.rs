use cocoa::appkit::{NSMainMenuWindowLevel, NSWindow};
use cocoa::base::id;
use tauri::{ActivationPolicy, App, Window};

pub fn platform(app: &mut App, main_window: Window, preference_window: Window) {
    // 磨砂窗口：https://github.com/tauri-apps/window-vibrancy
    window_vibrancy::apply_vibrancy(
        &preference_window,
        window_vibrancy::NSVisualEffectMaterial::Sidebar,
        Some(window_vibrancy::NSVisualEffectState::Active),
        Some(10.0),
    )
    .unwrap();

    // 隐藏 mac 的程序坞图标：https://github.com/tauri-apps/tauri/issues/4852#issuecomment-1312716378
    app.set_activation_policy(ActivationPolicy::Accessory);

    unsafe {
        let ns_window = main_window.ns_window().unwrap() as id;

        // 让窗口在程序坞和菜单栏之上
        ns_window.setLevel_(NSMainMenuWindowLevel as i64 + 1);
    }
}
