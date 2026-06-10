pub mod lifecycle;
pub(super) mod position;
pub mod preview;
mod state;

#[cfg(target_os = "macos")]
pub mod macos;
#[cfg(target_os = "windows")]
pub mod windows;

#[cfg(target_os = "macos")]
pub use macos::handle_reopen;
pub use state::WindowStateStore;

use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{LazyLock, Mutex};

use tauri::{AppHandle, Emitter, Manager, WebviewUrl, WebviewWindow, WebviewWindowBuilder, Window};

use crate::core::Result;
use crate::settings::{SettingsStore, WindowPosition};

pub const MAIN_WINDOW_LABEL: &str = "main";
pub const PREFERENCE_WINDOW_LABEL: &str = "preference";
pub const CLIPBOARD_PREVIEW_WINDOW_LABEL: &str = "clipboard-preview";

/// 偏好页定位高亮事件。前端收到后切到目标设置项所在分类并滚动高亮。
const PREFERENCE_HIGHLIGHT_EVENT: &str = "preference://highlight-setting";

/// 偏好窗口重建前暂存的高亮目标设置项。
///
/// preference 改为空闲可销毁后，「打开偏好并定位到某设置项」这类一次性投递存在竞态：
/// 窗口已销毁时重建是异步的，直接 `emit` 会丢给尚未挂载的前端（与 backup 接收同源）。
/// 故窗口不存在时先存入此 slot，由前端重建后经 `take_pending_preference_highlight` 主动拉取。
static PENDING_PREFERENCE_HIGHLIGHT: LazyLock<Mutex<Option<String>>> =
    LazyLock::new(|| Mutex::new(None));

#[derive(Clone, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct PreferenceHighlightPayload {
    setting_id: String,
}

/// 主窗口「固定」状态：true 时失焦不自动隐藏（点击窗外、切到其它 App 都不会隐藏），
/// 由前端 Pin 按钮 / 快捷键切换；macOS resign_key 与 Windows 外部点击钩子都尊重这个开关。
static MAIN_WINDOW_PINNED: AtomicBool = AtomicBool::new(false);
/// 主窗口自动隐藏的临时暂停状态，用于系统文件选择等会短暂转移焦点的原生交互。
static MAIN_WINDOW_AUTO_HIDE_SUSPENDED: AtomicBool = AtomicBool::new(false);

/// 返回用户是否显式固定主窗口；复制后隐藏等路径仍需读取这个用户态开关。
pub fn is_main_window_pinned() -> bool {
    MAIN_WINDOW_PINNED.load(Ordering::Relaxed)
}

/// 判断主窗口当前是否允许因失焦或外部点击自动隐藏。
pub fn should_auto_hide_main_window() -> bool {
    !MAIN_WINDOW_PINNED.load(Ordering::Relaxed)
        && !MAIN_WINDOW_AUTO_HIDE_SUSPENDED.load(Ordering::Relaxed)
}

/// 设置用户控制的主窗口固定态。
pub fn set_main_window_pinned(pinned: bool) {
    MAIN_WINDOW_PINNED.store(pinned, Ordering::Relaxed);
}

/// 临时暂停主窗口自动隐藏，不改变用户控制的固定态。
pub fn set_main_window_auto_hide_suspended(suspended: bool) {
    MAIN_WINDOW_AUTO_HIDE_SUSPENDED.store(suspended, Ordering::Relaxed);
}

/// 主窗口显隐变化事件。前端用以做默认聚焦 / 自动清空搜索等 UI 副作用。
/// 由 [`show_window`] / [`hide_window`] 在统一入口处发出，平台一致，
/// 不依赖 `tauri://focus` / `tauri://blur`（Windows 主窗口 `focusable: false` 不可靠）。
const WINDOW_VISIBILITY_EVENT: &str = "window://visibility";

#[derive(Clone, serde::Serialize)]
struct WindowVisibilityPayload<'a> {
    label: &'a str,
    visible: bool,
}

pub(super) fn emit_visibility(app_handle: &AppHandle, label: &str, visible: bool) {
    if let Err(err) = app_handle.emit(
        WINDOW_VISIBILITY_EVENT,
        WindowVisibilityPayload { label, visible },
    ) {
        log::error!("emit window visibility failed: {err:?}");
    }
}

pub(super) fn get_window(app_handle: &AppHandle, label: &str) -> Result<WebviewWindow> {
    app_handle
        .get_webview_window(label)
        .ok_or_else(|| anyhow::anyhow!("window not found: {label}").into())
}

pub fn show_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    // 销毁后重建：`DestroyWhenIdle` 窗口空闲超时后 WebView 已被销毁，打开时按 descriptor
    // 的 build fn 重新建窗。重建后窗口为 `visible: false`，下方走与既有一致的恢复 + show 流程。
    if app_handle.get_webview_window(label).is_none() {
        if let Some(build) = lifecycle::rebuild_fn(label) {
            build(app_handle)?;
        }
    }

    if label == MAIN_WINDOW_LABEL {
        if let Err(err) = apply_main_layout(app_handle) {
            log::warn!("apply main window layout failed: {err}");
        }
    } else {
        let visible = get_window(app_handle, label)?.is_visible().unwrap_or(false);

        if !visible {
            // 次级窗口（如 preference）：只在从隐藏态打开时恢复位置 + 尺寸。
            // 已可见窗口可能刚被用户移动但尚未落盘，重复恢复会把窗口拉回旧位置。
            if let Err(err) = state::restore_window_state(app_handle, label) {
                log::warn!("restore window state failed for {label}: {err}");
            }
        }
    }

    #[cfg(target_os = "macos")]
    let result = macos::show_window(app_handle, label);
    #[cfg(target_os = "windows")]
    let result = windows::show_window(app_handle, label);
    if result.is_ok() && !delays_main_visibility_event(label) {
        if label == MAIN_WINDOW_LABEL {
            preview::resume_after_main_show();
        }
        emit_visibility(app_handle, label, true);
        lifecycle::on_shown(app_handle, label);
    }
    result
}

/// macOS 主窗口有延迟 show，visibility 需等 NSPanel 真的显示后再 emit。
fn delays_main_visibility_event(label: &str) -> bool {
    cfg!(target_os = "macos") && label == MAIN_WINDOW_LABEL
}

pub fn hide_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    // 隐藏前保存任意窗口的实时几何：移动与缩放都在这里落盘，下次显示/启动可恢复。
    if let Err(err) = state::save_window_state(app_handle, label) {
        log::warn!("save window state on hide failed for {label}: {err}");
    }

    if label == MAIN_WINDOW_LABEL {
        preview::suppress_for_main_hide(app_handle);
    }

    #[cfg(target_os = "macos")]
    let result = macos::hide_window(app_handle, label);
    #[cfg(target_os = "windows")]
    let result = windows::hide_window(app_handle, label);
    if result.is_ok() {
        emit_visibility(app_handle, label, false);
        lifecycle::on_hidden(app_handle, label, "hide");
    }
    result
}

pub fn toggle_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    // 已销毁的按需窗口（如空闲超时后的 preference）取不到实例，视为不可见 → 走 show 重建。
    let visible = app_handle
        .get_webview_window(label)
        .and_then(|window| window.is_visible().ok())
        .unwrap_or(false);
    if visible {
        hide_window(app_handle, label)
    } else {
        show_window(app_handle, label)
    }
}

pub fn show_taskbar_icon(app_handle: &AppHandle, visible: bool) -> Result<()> {
    #[cfg(target_os = "macos")]
    return macos::show_taskbar_icon(app_handle, visible);
    #[cfg(target_os = "windows")]
    return windows::show_taskbar_icon(app_handle, visible);
}

pub fn position_window(app_handle: &AppHandle, label: &str, pos: WindowPosition) -> Result<()> {
    let window = get_window(app_handle, label)?;
    position::position_window(&window, pos)
}

/// 主窗显示前按设置应用窗口定位策略。
/// 始终先调用 `restore_window_state` 恢复尺寸与合法位置（含越界 fallback）；
/// 非 Remember 策略再由 `position_window` 覆盖位置。
/// 平台 `show_window` 需要在主线程闭包里调用，避免 set_position 与 show 异步交错产生闪烁。
fn apply_main_layout(app_handle: &AppHandle) -> Result<()> {
    let Some(store) = app_handle.try_state::<SettingsStore>() else {
        return Ok(());
    };
    let snap = store.snapshot();
    let position = snap.clipboard.window.position;

    let _ = state::restore_window_state(app_handle, MAIN_WINDOW_LABEL)?;

    if matches!(position, WindowPosition::Remember) {
        return Ok(());
    }

    let window = get_window(app_handle, MAIN_WINDOW_LABEL)?;
    position::position_window(&window, position)
}

/// 保存当前所有窗口的几何信息。供应用退出（`RunEvent::ExitRequested`）时调用，
/// 覆盖「调整大小后不关窗直接退出」这一隐藏/关闭都漏掉的场景。
pub fn save_all_window_states(app_handle: &AppHandle) {
    for label in app_handle.webview_windows().into_keys() {
        if let Err(err) = state::save_window_state(app_handle, &label) {
            log::warn!("save window state on exit failed for {label}: {err}");
        }
    }
}

/// 关闭请求改为隐藏窗口，让应用常驻后台（系统托盘）。
/// 返回 `true` 表示已拦截关闭，调用方需 `api.prevent_close()`。
///
/// 所有窗口的关闭按钮统一 hide，不直接销毁。`DestroyWhenIdle` 窗口（preference）
/// 在 hide 触发的 `on_hidden` 里启动空闲计时器，超时后才由生命周期管理器 `destroy`，
/// 故无需在 close 路径区分销毁分支。
pub fn hide_on_close(window: &Window) -> bool {
    // 关闭按钮不走 `hide_window`，需在此单独保存几何，否则 preference 的移动/缩放会丢失。
    if let Err(err) = state::save_window_state(window.app_handle(), window.label()) {
        log::warn!(
            "save window state on close failed for {}: {err}",
            window.label()
        );
    }

    if let Err(err) = window.hide() {
        log::error!("hide window on close failed: {err:?}");
    } else {
        emit_visibility(window.app_handle(), window.label(), false);
        lifecycle::on_hidden(window.app_handle(), window.label(), "close");
    }
    true
}

/// 按需重建 preference 窗口。preference 不再由 Tauri 配置预创建（改为 `DestroyWhenIdle`），
/// 故所有选项必须在此用 builder 完整复刻原 `tauri.conf.json` 声明，否则重建后行为漂移。
///
/// 建窗后保持 `visible: false`：由 [`show_window`] 统一走恢复几何 + 平台 show 流程，
/// 与其它窗口的显示路径一致。
pub fn build_preference_window(app_handle: &AppHandle) -> Result<()> {
    if app_handle
        .get_webview_window(PREFERENCE_WINDOW_LABEL)
        .is_some()
    {
        return Ok(());
    }

    let builder = WebviewWindowBuilder::new(
        app_handle,
        PREFERENCE_WINDOW_LABEL,
        WebviewUrl::App("index.html/#/preference".into()),
    )
    .title("EcoPaste Preference")
    .inner_size(960.0, 600.0)
    .min_inner_size(960.0, 600.0)
    .center()
    .maximizable(false)
    .skip_taskbar(true)
    .accept_first_mouse(true)
    .disable_drag_drop_handler()
    .visible(false);

    #[cfg(target_os = "macos")]
    let builder = builder
        .title_bar_style(tauri::TitleBarStyle::Overlay)
        .hidden_title(true);

    builder
        .build()
        .map_err(|err| anyhow::anyhow!("build preference window: {err}"))?;

    Ok(())
}

/// 打开偏好窗口并定位到指定设置项。
///
/// 偏好窗口存活时直接 emit 高亮事件；已空闲销毁时先把目标存入 pending slot，再 show
/// 触发重建——前端重建后经 [`take_pending_preference_highlight`] 主动拉取，规避
/// 「重建异步、push 丢失」竞态。所有「打开偏好并跳转某设置项」的入口都应走这里，
/// 不要在前端 `show_window` 后直接 `emitTo`。
pub fn open_preference_with_highlight(app_handle: &AppHandle, setting_id: String) -> Result<()> {
    let exists = app_handle
        .get_webview_window(PREFERENCE_WINDOW_LABEL)
        .is_some();

    if !exists {
        set_pending_preference_highlight(setting_id.clone());
    }

    show_window(app_handle, PREFERENCE_WINDOW_LABEL)?;

    if exists {
        app_handle
            .emit_to(
                PREFERENCE_WINDOW_LABEL,
                PREFERENCE_HIGHLIGHT_EVENT,
                PreferenceHighlightPayload { setting_id },
            )
            .map_err(|err| anyhow::anyhow!("emit preference highlight: {err}"))?;
    }

    Ok(())
}

/// 存入待定位的高亮目标，覆盖旧值（仅保留最近一次）。
fn set_pending_preference_highlight(setting_id: String) {
    let mut guard = PENDING_PREFERENCE_HIGHLIGHT
        .lock()
        .unwrap_or_else(|poisoned| {
            log::error!("pending preference highlight mutex poisoned on set, recovering");
            poisoned.into_inner()
        });
    *guard = Some(setting_id);
}

/// 取走并清空待定位的高亮目标，供偏好窗口重建后首屏拉取。
pub fn take_pending_preference_highlight() -> Option<String> {
    let mut guard = PENDING_PREFERENCE_HIGHLIGHT
        .lock()
        .unwrap_or_else(|poisoned| {
            log::error!("pending preference highlight mutex poisoned on take, recovering");
            poisoned.into_inner()
        });
    guard.take()
}
