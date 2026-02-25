use super::{shared_hide_window, shared_show_window};
use tauri::{command, AppHandle, Runtime, WebviewWindow};
use serde::Serialize;

#[derive(Serialize)]
pub struct CaretPosition {
    pub x: i32,
    pub y: i32,
    pub success: bool,
}

// 获取输入光标位置（Windows专用）
#[cfg(target_os = "windows")]
#[command]
pub fn get_caret_position() -> CaretPosition {
    use windows::Win32::Foundation::POINT;
    use windows::Win32::UI::WindowsAndMessaging::{
        GetCaretPos, GetForegroundWindow, GetWindowThreadProcessId,
        GetGUIThreadInfo, GUITHREADINFO,
    };
    use windows::Win32::System::Threading::{GetCurrentThreadId, AttachThreadInput};
    use windows::Win32::Graphics::Gdi::ClientToScreen;

    unsafe {
        let mut point = POINT { x: 0, y: 0 };
        
        // 获取前台窗口
        let foreground = GetForegroundWindow();
        if foreground.is_invalid() {
            return CaretPosition { x: 0, y: 0, success: false };
        }
        
        // 获取前台窗口线程ID
        let foreground_thread = GetWindowThreadProcessId(foreground, None);
        let current_thread = GetCurrentThreadId();
        
        // 附加线程输入以获取正确的光标位置
        let _ = AttachThreadInput(current_thread, foreground_thread, true);
        
        let mut success = false;

        // 尝试方法 1: GetCaretPos
        if GetCaretPos(&mut point).is_ok() && (point.x != 0 || point.y != 0) {
            success = true;
        } 
        
        // 尝试方法 2: GetGUIThreadInfo (如果在方法 1 失败或返回 0,0 时)
        if !success {
            let mut gui_info = GUITHREADINFO::default();
            gui_info.cbSize = std::mem::size_of::<GUITHREADINFO>() as u32;
            
            if GetGUIThreadInfo(foreground_thread, &mut gui_info).is_ok() {
                // rcCaret 是相对于 carets 所在窗口的客户区坐标
                // 如果 rcCaret 有效
                if gui_info.rcCaret.right > gui_info.rcCaret.left {
                    point.x = gui_info.rcCaret.left;
                    point.y = gui_info.rcCaret.bottom;
                    success = true;
                    // 注意：GetGUIThreadInfo 的 rcCaret 坐标是基于 hwndCaret 的
                    // 我们需要将它转换到屏幕坐标
                    if !gui_info.hwndCaret.is_invalid() {
                         let _ = ClientToScreen(gui_info.hwndCaret, &mut point);
                         // ClientToScreen 已经转换了 point，我们不需要再做下面的 ClientToScreen(foreground)
                         // 但我们需要分离线程输入并返回
                         let _ = AttachThreadInput(current_thread, foreground_thread, false);
                         return CaretPosition { x: point.x, y: point.y, success: true };
                    }
                }
            }
        }
        
        // 分离线程输入
        let _ = AttachThreadInput(current_thread, foreground_thread, false);
        
        if success {
            // 转换为屏幕坐标（使用前台窗口，如果是 GetCaretPos 获取的）
            let _ = ClientToScreen(foreground, &mut point);
            CaretPosition { x: point.x, y: point.y, success: true }
        } else {
            CaretPosition { x: 0, y: 0, success: false }
        }
    }
}

#[cfg(not(target_os = "windows"))]
#[command]
pub fn get_caret_position() -> CaretPosition {
    CaretPosition { x: 0, y: 0, success: false }
}

// 显示窗口
#[command]
pub async fn show_window<R: Runtime>(_app_handle: AppHandle<R>, window: WebviewWindow<R>) {
    shared_show_window(&window);
}

// 隐藏窗口
#[command]
pub async fn hide_window<R: Runtime>(_app_handle: AppHandle<R>, window: WebviewWindow<R>) {
    shared_hide_window(&window);
}

// 显示任务栏图标
#[command]
pub async fn show_taskbar_icon<R: Runtime>(
    _app_handle: AppHandle<R>,
    window: WebviewWindow<R>,
    visible: bool,
) {
    let _ = window.set_skip_taskbar(!visible);
}

