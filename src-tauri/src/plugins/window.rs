use tauri::{
    command, generate_handler,
    plugin::{Builder, TauriPlugin},
    utils::config::WindowConfig,
    AppHandle, Manager, Window, WindowBuilder, Wry,
};
use windows::Win32::Foundation::HWND;
use windows::Win32::UI::WindowsAndMessaging::{
    GetWindowLongPtrA, SetWindowLongPtrA, SetWindowPos, GWL_EXSTYLE, HWND_TOPMOST,
    SWP_FRAMECHANGED, SWP_NOMOVE, SWP_NOSIZE, WINDOW_EX_STYLE, WS_EX_ACCEPTFILES,
    WS_EX_APPWINDOW, WS_EX_NOACTIVATE, WS_EX_TOOLWINDOW,
};

// 主窗口的名称
pub static MAIN_WINDOW_LABEL: &str = "main";

// 创建窗口
#[command]
pub async fn create_window(app_handle: AppHandle, label: String, mut options: WindowConfig) {
    if let Some(window) = app_handle.get_window(&label) {
        show_window(window).await;
    } else {
        options.label = label.to_string();

        let window = WindowBuilder::from_config(&app_handle, options.clone())
            .build()
            .unwrap();

        if !options.decorations {
            window_shadows::set_shadow(&window, true).unwrap();
        }
    }
}

// 显示窗口
#[command]
pub async fn show_window(window: Window) {
    window.show().unwrap();
    window.unminimize().unwrap();

    if window.label().eq("clipboard-history") {
        noactivate(window);
    } else {
        window.set_focus().unwrap();
    }
}

// 隐藏窗口
#[command]
pub async fn hide_window(window: Window) {
    window.hide().unwrap();
}

// 退出 app
#[command]
pub async fn quit_app() {
    std::process::exit(0)
}

#[command]
pub fn noactivate(window: Window) {
    #[cfg(target_os = "windows")]
    {
        println!("window label: {}",window.label());
        let hwnd = window.hwnd().unwrap();
        unsafe {
            // 经测试,每次show的时候窗口扩展样式会被tauri重置.所以不用加只执行一次的判断.
            // 有俩个windows-rs库,显式转换一下HWND!!!
            let hwnd = HWND(hwnd.0);
            let style = GetWindowLongPtrA(hwnd, GWL_EXSTYLE);
            let mut style = WINDOW_EX_STYLE(style as u32);
            if !style.contains(WS_EX_NOACTIVATE) {
                style &= !WS_EX_APPWINDOW;
                style &= !WS_EX_ACCEPTFILES;
                let _ = SetWindowLongPtrA(
                    hwnd,
                    GWL_EXSTYLE,
                    (style | WS_EX_NOACTIVATE | WS_EX_TOOLWINDOW).0 as isize,
                );
                let uflags = SWP_NOMOVE | SWP_NOSIZE | SWP_FRAMECHANGED; //SWP_NOZORDER |
                SetWindowPos(hwnd, HWND_TOPMOST, 0, 0, 0, 0, uflags).unwrap();
            }
        }
    }
}

#[command]
pub fn activate(window: Window) {
    #[cfg(target_os = "windows")]
    {
        println!("window label: {}",window.label());
        let hwnd = window.hwnd().unwrap();
        unsafe {
            // 经测试,每次show的时候窗口扩展样式会被tauri重置.所以不用加只执行一次的判断.
            // 有俩个windows-rs库,显式转换一下HWND!!!
            let hwnd = HWND(hwnd.0);
            let style = GetWindowLongPtrA(hwnd, GWL_EXSTYLE);
            let mut style = WINDOW_EX_STYLE(style as u32);
            if style.contains(WS_EX_NOACTIVATE) {
                style &= !WS_EX_APPWINDOW;
                style &= !WS_EX_ACCEPTFILES;
                style &= !WS_EX_NOACTIVATE;
                let _ = SetWindowLongPtrA(
                    hwnd,
                    GWL_EXSTYLE,
                    style.0 as isize,
                );
                let uflags = SWP_NOMOVE | SWP_NOSIZE | SWP_FRAMECHANGED; //SWP_NOZORDER |
                SetWindowPos(hwnd, HWND_TOPMOST, 0, 0, 0, 0, uflags).unwrap();
            }
        }
    }
}

pub fn init() -> TauriPlugin<Wry> {
    Builder::new("window")
        .invoke_handler(generate_handler![
            create_window,
            show_window,
            hide_window,
            quit_app,
            activate,
            noactivate
        ])
        .build()
}