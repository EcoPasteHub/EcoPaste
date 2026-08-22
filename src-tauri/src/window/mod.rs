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

use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::{LazyLock, Mutex};
use std::time::Duration;

use tauri::{AppHandle, Emitter, Manager, WebviewUrl, WebviewWindow, WebviewWindowBuilder, Window};

use crate::core::Result;
use crate::settings::{SettingsStore, WindowPosition};

pub const CLIPBOARD_WINDOW_LABEL: &str = "clipboard";
/// Saved geometry for the bottom history shelf; kept separate from the floating panel size.
pub const CLIPBOARD_DOCK_STATE_LABEL: &str = "clipboard-dock";
pub const PREFERENCE_WINDOW_LABEL: &str = "preference";
pub const CLIPBOARD_PREVIEW_WINDOW_LABEL: &str = "clipboard-preview";
pub const ONBOARDING_WINDOW_LABEL: &str = "onboarding";
pub const UPDATE_WINDOW_LABEL: &str = "update";

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

/// 剪贴板窗口「固定」状态：true 时失焦不自动隐藏（点击窗外、切到其它 App 都不会隐藏），
/// 由前端 Pin 按钮 / 快捷键切换；macOS resign_key 与 Windows 外部点击钩子都尊重这个开关。
static CLIPBOARD_WINDOW_PINNED: AtomicBool = AtomicBool::new(false);
/// 剪贴板窗口自动隐藏的临时暂停状态，用于系统文件选择等会短暂转移焦点的原生交互。
static CLIPBOARD_WINDOW_AUTO_HIDE_SUSPENDED: AtomicBool = AtomicBool::new(false);
/// Prevent resize/move snapping from recursively applying dock geometry.
static SNAPPING_CLIPBOARD_DOCK: AtomicBool = AtomicBool::new(false);
/// Dock shelf hide animation: bumped on show so an in-flight slide-down can cancel.
static DOCK_HIDE_GENERATION: AtomicU64 = AtomicU64::new(0);
static DOCK_HIDE_PENDING: AtomicBool = AtomicBool::new(false);
const DOCK_SLIDE_HIDE_MS: u64 = 320;
const WINDOW_PREPARE_HIDE_EVENT: &str = "window://prepare-hide";

/// 返回用户是否显式固定剪贴板窗口；复制后隐藏等路径仍需读取这个用户态开关。
pub fn is_clipboard_window_pinned() -> bool {
    CLIPBOARD_WINDOW_PINNED.load(Ordering::Relaxed)
}

/// 判断剪贴板窗口当前是否允许因失焦或外部点击自动隐藏。
pub fn should_auto_hide_clipboard_window() -> bool {
    !CLIPBOARD_WINDOW_PINNED.load(Ordering::Relaxed)
        && !CLIPBOARD_WINDOW_AUTO_HIDE_SUSPENDED.load(Ordering::Relaxed)
}

/// 设置用户控制的剪贴板窗口固定态。
pub fn set_clipboard_window_pinned(pinned: bool) {
    CLIPBOARD_WINDOW_PINNED.store(pinned, Ordering::Relaxed);
}

/// 临时暂停剪贴板窗口自动隐藏，不改变用户控制的固定态。
pub fn set_clipboard_window_auto_hide_suspended(suspended: bool) {
    CLIPBOARD_WINDOW_AUTO_HIDE_SUSPENDED.store(suspended, Ordering::Relaxed);
}

pub fn set_clipboard_window_editing(app_handle: &AppHandle, editing: bool) -> Result<()> {
    #[cfg(target_os = "windows")]
    return windows::set_clipboard_window_editing(app_handle, editing);

    #[cfg(target_os = "macos")]
    {
        let _ = app_handle;
        let _ = editing;

        Ok(())
    }
}

/// 剪贴板窗口显隐变化事件。前端用以做默认聚焦 / 自动清空搜索等 UI 副作用。
/// 由 [`show_window`] / [`hide_window`] 在统一入口处发出，平台一致，
/// 不依赖 `tauri://focus` / `tauri://blur`（Windows 剪贴板窗口 `focusable: false` 不可靠）。
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

    if label == CLIPBOARD_WINDOW_LABEL {
        cancel_dock_slide_hide();
        if let Err(err) = apply_clipboard_window_layout(app_handle) {
            log::warn!("apply clipboard window layout failed: {err}");
        }
    } else if label == ONBOARDING_WINDOW_LABEL {
        if let Err(err) = position_window(app_handle, label, WindowPosition::Center) {
            log::warn!("center onboarding window failed: {err}");
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
    if result.is_ok() && !delays_clipboard_visibility_event(label) {
        if label == CLIPBOARD_WINDOW_LABEL {
            preview::resume_after_clipboard_show();
        }
        emit_visibility(app_handle, label, true);
        lifecycle::on_shown(app_handle, label);
    }
    result
}

/// macOS 剪贴板窗口有延迟 show，visibility 需等 NSPanel 真的显示后再 emit。
fn delays_clipboard_visibility_event(label: &str) -> bool {
    cfg!(target_os = "macos") && label == CLIPBOARD_WINDOW_LABEL
}

pub fn hide_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    // 隐藏前保存任意窗口的实时几何：移动与缩放都在这里落盘，下次显示/启动可恢复。
    if let Err(err) = save_window_geometry(app_handle, label) {
        log::warn!("save window state on hide failed for {label}: {err}");
    }

    if label == CLIPBOARD_WINDOW_LABEL {
        preview::suppress_for_clipboard_hide(app_handle);
        if clipboard_uses_dock(app_handle) {
            schedule_dock_slide_hide(app_handle.clone());
            return Ok(());
        }
    }

    finish_hide_window(app_handle, label)
}

fn finish_hide_window(app_handle: &AppHandle, label: &str) -> Result<()> {
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

fn clipboard_uses_dock(app_handle: &AppHandle) -> bool {
    app_handle
        .try_state::<SettingsStore>()
        .is_some_and(|store| store.snapshot().clipboard.window.position.is_bottom_dock())
}

fn cancel_dock_slide_hide() {
    DOCK_HIDE_GENERATION.fetch_add(1, Ordering::SeqCst);
    DOCK_HIDE_PENDING.store(false, Ordering::SeqCst);
}

/// Let the frontend slide the shelf down, then hide the NSPanel/window.
fn schedule_dock_slide_hide(app_handle: AppHandle) {
    if DOCK_HIDE_PENDING.swap(true, Ordering::SeqCst) {
        return;
    }

    let generation = DOCK_HIDE_GENERATION.load(Ordering::SeqCst);
    if let Err(err) = app_handle.emit(
        WINDOW_PREPARE_HIDE_EVENT,
        WindowVisibilityPayload {
            label: CLIPBOARD_WINDOW_LABEL,
            visible: false,
        },
    ) {
        log::warn!("emit window prepare-hide failed: {err:?}");
    }

    tauri::async_runtime::spawn(async move {
        tokio::time::sleep(Duration::from_millis(DOCK_SLIDE_HIDE_MS)).await;

        if DOCK_HIDE_GENERATION.load(Ordering::SeqCst) != generation {
            return;
        }

        DOCK_HIDE_PENDING.store(false, Ordering::SeqCst);
        if let Err(err) = finish_hide_window(&app_handle, CLIPBOARD_WINDOW_LABEL) {
            log::warn!("finish dock slide hide failed: {err}");
        }
    });
}

pub fn toggle_window(app_handle: &AppHandle, label: &str) -> Result<()> {
    if label == CLIPBOARD_WINDOW_LABEL && DOCK_HIDE_PENDING.load(Ordering::SeqCst) {
        return show_window(app_handle, label);
    }

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

/// 剪贴板窗口显示前按设置应用窗口定位策略。
/// 面板模式先恢复存档尺寸与合法位置（含越界 fallback）；
/// 底部 shelf 使用独立存档高度，避免和浮动面板互相覆盖。
/// 平台 `show_window` 需要在主线程闭包里调用，避免 set_position 与 show 异步交错产生闪烁。
pub fn apply_clipboard_window_layout(app_handle: &AppHandle) -> Result<()> {
    let Some(store) = app_handle.try_state::<SettingsStore>() else {
        return Ok(());
    };
    if app_handle
        .get_webview_window(CLIPBOARD_WINDOW_LABEL)
        .is_none()
    {
        return Ok(());
    }
    let snap = store.snapshot();
    let position = snap.clipboard.window.position;
    let window = get_window(app_handle, CLIPBOARD_WINDOW_LABEL)?;

    if position.is_bottom_dock() {
        let dock_height = state::get_window_state(app_handle, CLIPBOARD_DOCK_STATE_LABEL)
            .map(|saved| saved.height);
        return position::apply_bottom_dock(&window, dock_height);
    }

    position::apply_panel_constraints(&window)?;
    let _ = state::restore_window_state(app_handle, CLIPBOARD_WINDOW_LABEL)?;

    if matches!(position, WindowPosition::Remember) {
        return Ok(());
    }

    position::position_window(&window, position)
}

/// Keep a visible bottom shelf snapped to the work-area bottom after the user resizes or moves it.
pub fn snap_clipboard_dock_if_needed(app_handle: &AppHandle) {
    let Some(store) = app_handle.try_state::<SettingsStore>() else {
        return;
    };
    if !store.snapshot().clipboard.window.position.is_bottom_dock() {
        return;
    }
    let Some(window) = app_handle.get_webview_window(CLIPBOARD_WINDOW_LABEL) else {
        return;
    };
    if SNAPPING_CLIPBOARD_DOCK.swap(true, Ordering::SeqCst) {
        return;
    }
    let result = (|| -> Result<()> {
        let height = window
            .inner_size()
            .map_err(|err| anyhow::anyhow!(err))?
            .height;

        position::apply_bottom_dock(&window, Some(height))
    })();
    SNAPPING_CLIPBOARD_DOCK.store(false, Ordering::SeqCst);
    if let Err(err) = result {
        log::warn!("snap clipboard dock failed: {err}");
    }
}

/// Persist clipboard geometry into the panel or dock slot that matches the current layout.
pub fn save_clipboard_window_state(app_handle: &AppHandle) -> Result<()> {
    let position = app_handle
        .try_state::<SettingsStore>()
        .map(|store| store.snapshot().clipboard.window.position)
        .unwrap_or_default();
    let label = if position.is_bottom_dock() {
        CLIPBOARD_DOCK_STATE_LABEL
    } else {
        CLIPBOARD_WINDOW_LABEL
    };

    state::save_window_state_as(app_handle, CLIPBOARD_WINDOW_LABEL, label)
}

/// 保存当前所有窗口的几何信息。供应用退出（`RunEvent::ExitRequested`）时调用，
/// 覆盖「调整大小后不关窗直接退出」这一隐藏/关闭都漏掉的场景。
pub fn save_all_window_states(app_handle: &AppHandle) {
    for label in app_handle.webview_windows().into_keys() {
        if let Err(err) = save_window_geometry(app_handle, &label) {
            log::warn!("save window state on exit failed for {label}: {err}");
        }
    }
}

fn save_window_geometry(app_handle: &AppHandle, label: &str) -> Result<()> {
    if label == CLIPBOARD_WINDOW_LABEL {
        return save_clipboard_window_state(app_handle);
    }

    state::save_window_state(app_handle, label)
}

/// 处理窗口关闭请求，让应用常驻后台（系统托盘）。
/// 返回 `true` 表示已拦截关闭，调用方需 `api.prevent_close()`。
///
/// 引导窗口属于强制流程，关闭请求只拦截不隐藏；其它窗口的关闭按钮统一 hide，不直接销毁。
/// `DestroyWhenIdle` 窗口在 hide 触发的 `on_hidden` 里启动空闲计时器，超时后才由生命周期
/// 管理器 `destroy`，故无需在 close 路径区分销毁分支。
pub fn intercept_close_request(window: &Window) -> bool {
    if window.label() == ONBOARDING_WINDOW_LABEL {
        return true;
    }

    if window.label() == CLIPBOARD_WINDOW_LABEL {
        if let Err(err) = hide_window(window.app_handle(), CLIPBOARD_WINDOW_LABEL) {
            log::warn!("hide clipboard window on close failed: {err}");
        }
        return true;
    }

    // 关闭按钮不走 `hide_window`，需在此单独保存几何，否则 preference 的移动/缩放会丢失。
    if let Err(err) = save_window_geometry(window.app_handle(), window.label()) {
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

/// 按需创建软件更新窗口。更新流程由 Rust updater 命令驱动，窗口只负责渲染状态。
pub fn build_update_window(app_handle: &AppHandle) -> Result<()> {
    if app_handle.get_webview_window(UPDATE_WINDOW_LABEL).is_some() {
        return Ok(());
    }

    let builder = WebviewWindowBuilder::new(
        app_handle,
        UPDATE_WINDOW_LABEL,
        WebviewUrl::App("index.html/#/update".into()),
    )
    .title("EcoPaste Update")
    .inner_size(520.0, 230.0)
    .min_inner_size(520.0, 230.0)
    .center()
    .maximizable(false)
    .resizable(false)
    .skip_taskbar(true)
    .accept_first_mouse(true)
    .disable_drag_drop_handler()
    .decorations(true)
    .transparent(false)
    .visible(false);

    #[cfg(target_os = "macos")]
    let builder = builder
        .title_bar_style(tauri::TitleBarStyle::Overlay)
        .hidden_title(true);

    builder
        .build()
        .map_err(|err| anyhow::anyhow!("build update window: {err}"))?;

    Ok(())
}

/// 按需创建首次启动引导窗口。引导窗口始终无边框、深色 UI、打开时居中。
pub fn build_onboarding_window(app_handle: &AppHandle) -> Result<()> {
    if app_handle
        .get_webview_window(ONBOARDING_WINDOW_LABEL)
        .is_some()
    {
        return Ok(());
    }

    WebviewWindowBuilder::new(
        app_handle,
        ONBOARDING_WINDOW_LABEL,
        WebviewUrl::App("index.html/#/onboarding".into()),
    )
    .title("EcoPaste Onboarding")
    .inner_size(900.0, 600.0)
    .center()
    .resizable(false)
    .maximizable(false)
    .decorations(false)
    .transparent(true)
    .accept_first_mouse(true)
    .disable_drag_drop_handler()
    .visible(false)
    .build()
    .map_err(|err| anyhow::anyhow!("build onboarding window: {err}"))?;

    Ok(())
}

/// 创建并显示首次启动引导窗口。
pub fn open_onboarding(app_handle: &AppHandle) -> Result<()> {
    if app_handle
        .get_webview_window(ONBOARDING_WINDOW_LABEL)
        .is_none()
    {
        build_onboarding_window(app_handle)?;
    }

    show_window(app_handle, ONBOARDING_WINDOW_LABEL)
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
