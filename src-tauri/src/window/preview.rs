//! 系统级剪贴板预览窗口骨架。
//!
//! 预览窗口是透明的 full-screen overlay：Rust 负责按需建窗、复用窗口、
//! 收集坐标上下文并广播，前端在该窗口内渲染预览内容与连接曲线。
//! 窗口接入生命周期管理（`DestroyWhenIdle`）：隐藏空闲后销毁 WebView 释放内存，
//! 仅在预览请求到达时按需建窗，不随主窗口显示预热。

#![allow(clippy::unused_unit)]

use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::{LazyLock, Mutex};
use std::thread;
use std::time::Duration;

use serde::{Deserialize, Serialize};
use tauri::{
    AppHandle, Emitter, Manager, PhysicalPosition, PhysicalRect, PhysicalSize, WebviewUrl,
    WebviewWindow, WebviewWindowBuilder,
};

use crate::core::Result;

use super::{get_window, lifecycle, CLIPBOARD_PREVIEW_WINDOW_LABEL, MAIN_WINDOW_LABEL};

#[cfg(target_os = "macos")]
use tauri_nspanel::{tauri_panel, ManagerExt, PanelLevel, WebviewWindowExt};

#[cfg(target_os = "windows")]
use windows::Win32::Foundation::HWND;
#[cfg(target_os = "windows")]
use windows::Win32::UI::WindowsAndMessaging::{
    SetWindowPos, HWND_TOPMOST, SWP_NOACTIVATE, SWP_NOMOVE, SWP_NOSIZE, SWP_SHOWWINDOW,
};

const PREVIEW_UPDATED_EVENT: &str = "preview://updated";
const PREVIEW_PANEL_WIDTH: f64 = 480.0;
const PREVIEW_PANEL_HEIGHT: f64 = 480.0;
const PREVIEW_PANEL_GAP: f64 = 40.0;
const PREVIEW_PANEL_MARGIN: f64 = 32.0;
const PREVIEW_POINTER_ANCHOR_SIZE: f64 = 1.0;
const PREVIEW_HIDE_DELAY_MS: u64 = 180;

static PREVIEW_REQUEST_ID: AtomicU64 = AtomicU64::new(0);
static PREVIEW_SESSION_ID: AtomicU64 = AtomicU64::new(0);
static PREVIEW_SUPPRESSED: AtomicBool = AtomicBool::new(false);
static PREVIEW_STATE: LazyLock<Mutex<Option<ClipboardPreviewState>>> =
    LazyLock::new(|| Mutex::new(None));
/// 串行化建窗：多个预览请求（如连续 hover）可能并发走到「检查不存在 → 建窗」，
/// 都过了存在性检查会触发重复 label 建窗报错。建窗都来自命令/后台线程、主线程从不持锁，
/// 不会与 builder 内部的主线程派发互锁。
static PREVIEW_BUILD_LOCK: Mutex<()> = Mutex::new(());

#[cfg(target_os = "macos")]
tauri_panel! {
    panel!(PreviewPanel {
        config: {
            is_floating_panel: true,
            can_become_key_window: false,
            can_become_main_window: false
        }
    })
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PreviewAnchorRect {
    pub left: f64,
    pub top: f64,
    pub width: f64,
    pub height: f64,
    pub pointer_y: Option<f64>,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PreviewWorkArea {
    pub x: i32,
    pub y: i32,
    pub width: u32,
    pub height: u32,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PreviewMainWindowRect {
    pub x: i32,
    pub y: i32,
    pub width: u32,
    pub height: u32,
}

#[derive(Clone, Copy, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PreviewRect {
    pub left: f64,
    pub top: f64,
    pub width: f64,
    pub height: f64,
}

#[derive(Clone, Copy, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub enum PreviewPlacement {
    Right,
    Left,
    Bottom,
    Top,
}

#[derive(Clone, Copy, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PreviewLayout {
    pub overlay_rect: PreviewRect,
    pub source_rect: PreviewRect,
    pub panel_rect: PreviewRect,
    pub placement: PreviewPlacement,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ClipboardPreviewState {
    pub request_id: u64,
    pub session_id: u64,
    pub item_id: String,
    pub anchor: PreviewAnchorRect,
    pub scale_factor: f64,
    pub work_area: PreviewWorkArea,
    pub main_window: Option<PreviewMainWindowRect>,
    pub layout: PreviewLayout,
}

/// 打开或重定向预览窗口，并把最新预览状态广播到预览 webview。
pub fn show_clipboard_preview(
    app: &AppHandle,
    item_id: String,
    anchor: PreviewAnchorRect,
) -> Result<Option<ClipboardPreviewState>> {
    validate_anchor(&anchor)?;

    if PREVIEW_SUPPRESSED.load(Ordering::SeqCst) || !is_main_window_visible(app) {
        close_clipboard_preview_now(app)?;
        return Ok(None);
    }

    let request_id = PREVIEW_REQUEST_ID.fetch_add(1, Ordering::SeqCst) + 1;
    let session_id = preview_session_id_for_show();
    let window = ensure_preview_window(app)?;
    let monitor = resolve_preview_monitor(app)?;
    let work_area = preview_overlay_bounds(&monitor);
    let scale_factor = monitor.scale_factor();
    let main_window = main_window_rect(app);
    let layout = build_preview_layout(&anchor, scale_factor, &work_area, main_window.as_ref());

    prepare_preview_window_for_show(app, &window, &work_area)?;

    let state = ClipboardPreviewState {
        request_id,
        session_id,
        item_id,
        anchor,
        scale_factor,
        work_area: PreviewWorkArea {
            x: work_area.position.x,
            y: work_area.position.y,
            width: work_area.size.width,
            height: work_area.size.height,
        },
        layout,
        main_window,
    };

    set_preview_state(Some(state.clone()));
    window
        .emit(PREVIEW_UPDATED_EVENT, &state)
        .map_err(|e| anyhow::anyhow!(e))?;
    show_preview_window(app, &window, &work_area)?;

    Ok(Some(state))
}

/// 隐藏预览窗口并清空当前预览状态。
pub fn close_clipboard_preview(app: &AppHandle) -> Result<()> {
    let request_id = PREVIEW_REQUEST_ID.fetch_add(1, Ordering::SeqCst) + 1;
    set_preview_state(None);

    if let Some(window) = app.get_webview_window(CLIPBOARD_PREVIEW_WINDOW_LABEL) {
        window
            .emit(PREVIEW_UPDATED_EVENT, Option::<ClipboardPreviewState>::None)
            .map_err(|e| anyhow::anyhow!(e))?;
        schedule_preview_window_hide(app.clone(), window, request_id);
    }

    Ok(())
}

/// 立即隐藏预览窗口并清空状态；用于主窗口隐藏等不需要退出动画的路径。
pub fn close_clipboard_preview_now(app: &AppHandle) -> Result<()> {
    PREVIEW_REQUEST_ID.fetch_add(1, Ordering::SeqCst);
    set_preview_state(None);

    if let Some(window) = app.get_webview_window(CLIPBOARD_PREVIEW_WINDOW_LABEL) {
        window
            .emit(PREVIEW_UPDATED_EVENT, Option::<ClipboardPreviewState>::None)
            .map_err(|e| anyhow::anyhow!(e))?;
        hide_preview_window(app, &window)?;
    }

    Ok(())
}

/// 主窗口开始隐藏时压制后续过期 show 请求，并立即收起预览窗口。
pub fn suppress_for_main_hide(app: &AppHandle) {
    PREVIEW_SUPPRESSED.store(true, Ordering::SeqCst);
    if let Err(error) = close_clipboard_preview_now(app) {
        log::error!("suppress preview on main hide failed: {error}");
    }
}

/// 主窗口重新显示后允许新的预览请求进入。预览窗口不随主窗口预创建，
/// 由首次 [`show_clipboard_preview`] 经 `ensure_preview_window` 按需建窗。
pub fn resume_after_main_show() {
    PREVIEW_SUPPRESSED.store(false, Ordering::SeqCst);
}

/// 返回预览窗口最近一次收到的状态，供预览页首屏补拉。
pub fn get_clipboard_preview_state() -> Result<Option<ClipboardPreviewState>> {
    let guard = PREVIEW_STATE.lock().unwrap_or_else(|poisoned| {
        log::error!("preview state mutex poisoned on get, recovering");
        poisoned.into_inner()
    });

    Ok(guard.clone())
}

/// 按需重建预览窗口。预览窗口不再由 Tauri 配置预创建（改为空闲销毁 + 按需重建），
/// 所有选项必须在此用 builder 完整复刻原 `tauri.conf.json` 声明，否则重建后行为漂移。
///
/// 建窗后保持 `visible: false`：定位与显示由预览 show 流程统一处理；
/// macOS 的 NSPanel 转换由 [`ensure_preview_window`] 在每次取窗时兜底执行。
pub fn build_clipboard_preview_window(app: &AppHandle) -> Result<()> {
    let _guard = PREVIEW_BUILD_LOCK.lock().unwrap_or_else(|poisoned| {
        log::error!("preview build mutex poisoned, recovering");
        poisoned.into_inner()
    });

    if app
        .get_webview_window(CLIPBOARD_PREVIEW_WINDOW_LABEL)
        .is_some()
    {
        return Ok(());
    }

    WebviewWindowBuilder::new(
        app,
        CLIPBOARD_PREVIEW_WINDOW_LABEL,
        WebviewUrl::App("index.html/#/preview".into()),
    )
    .title("EcoPaste Preview")
    .inner_size(1.0, 1.0)
    .resizable(false)
    .maximizable(false)
    .minimizable(false)
    .always_on_top(true)
    .decorations(false)
    .shadow(false)
    .transparent(true)
    .skip_taskbar(true)
    .focused(false)
    .focusable(false)
    .disable_drag_drop_handler()
    .visible(false)
    .build()
    .map_err(|err| anyhow::anyhow!("build clipboard preview window: {err}"))?;

    Ok(())
}

fn set_preview_state(state: Option<ClipboardPreviewState>) {
    let mut guard = PREVIEW_STATE.lock().unwrap_or_else(|poisoned| {
        log::error!("preview state mutex poisoned on set, recovering");
        poisoned.into_inner()
    });
    *guard = state;
}

/// 判断主窗口是否仍处于可见状态，防止过期 hover 请求在主窗口隐藏后唤起预览。
fn is_main_window_visible(app: &AppHandle) -> bool {
    app.get_webview_window(MAIN_WINDOW_LABEL)
        .and_then(|window| window.is_visible().ok())
        .unwrap_or(false)
}

/// 延迟隐藏真实窗口，为前端退出动画留出一小段可见时间。
fn schedule_preview_window_hide(app: AppHandle, window: WebviewWindow, request_id: u64) {
    thread::spawn(move || {
        thread::sleep(Duration::from_millis(PREVIEW_HIDE_DELAY_MS));

        if PREVIEW_REQUEST_ID.load(Ordering::SeqCst) != request_id {
            return;
        }
        if get_clipboard_preview_state().ok().flatten().is_some() {
            return;
        }

        if let Err(error) = hide_preview_window(&app, &window) {
            log::error!("hide preview window after exit animation failed: {error}");
        }
    });
}

/// 返回本次 show 所属的可见会话 id；隐藏后再次 show 会开启新会话。
fn preview_session_id_for_show() -> u64 {
    let guard = PREVIEW_STATE.lock().unwrap_or_else(|poisoned| {
        log::error!("preview state mutex poisoned on session, recovering");
        poisoned.into_inner()
    });

    if guard.is_some() {
        return PREVIEW_SESSION_ID.load(Ordering::SeqCst);
    }

    PREVIEW_SESSION_ID.fetch_add(1, Ordering::SeqCst) + 1
}

fn validate_anchor(anchor: &PreviewAnchorRect) -> Result<()> {
    let values = [anchor.left, anchor.top, anchor.width, anchor.height];
    if !values.iter().all(|value| value.is_finite()) || anchor.width <= 0.0 || anchor.height <= 0.0
    {
        return Err(anyhow::anyhow!("preview anchor is invalid").into());
    }
    if let Some(pointer_y) = anchor.pointer_y {
        if !pointer_y.is_finite() {
            return Err(anyhow::anyhow!("preview pointer is invalid").into());
        }
    }

    Ok(())
}

/// 取预览窗口；已被空闲销毁（或尚未创建）时按需重建。
/// macOS 下每次都兜底确保 NSPanel 转换完成，覆盖重建后的全新窗口。
fn ensure_preview_window(app: &AppHandle) -> Result<WebviewWindow> {
    if app
        .get_webview_window(CLIPBOARD_PREVIEW_WINDOW_LABEL)
        .is_none()
    {
        build_clipboard_preview_window(app)?;
    }

    let window = get_window(app, CLIPBOARD_PREVIEW_WINDOW_LABEL)?;

    #[cfg(target_os = "macos")]
    ensure_macos_preview_panel(app, &window)?;

    Ok(window)
}

fn raise_preview_window(app: &AppHandle, window: &WebviewWindow) -> Result<()> {
    #[cfg(target_os = "macos")]
    {
        let _ = window;
        set_macos_preview_panel_level(app)
    }

    #[cfg(target_os = "windows")]
    {
        let _ = app;
        window
            .set_always_on_top(true)
            .map_err(|e| anyhow::anyhow!(e))?;
        raise_windows_preview_window(window, false)?;

        Ok(())
    }
}

fn prepare_preview_window_for_show(
    app: &AppHandle,
    window: &WebviewWindow,
    work_area: &PhysicalRect<i32, u32>,
) -> Result<()> {
    apply_preview_window_bounds(window, work_area)?;
    window
        .set_ignore_cursor_events(true)
        .map_err(|e| anyhow::anyhow!(e))?;
    raise_preview_window(app, window)
}

/// 平台 show 收口点；成功后推进生命周期到 `Visible`，使未触发的空闲销毁计时器过期。
fn show_preview_window(
    app: &AppHandle,
    window: &WebviewWindow,
    work_area: &PhysicalRect<i32, u32>,
) -> Result<()> {
    #[cfg(target_os = "macos")]
    {
        let _ = window;
        show_macos_preview_panel(app, work_area)?;
    }

    #[cfg(target_os = "windows")]
    {
        let _ = work_area;
        window.show().map_err(|e| anyhow::anyhow!(e))?;
        raise_windows_preview_window(window, true)?;
    }

    lifecycle::on_shown(app, CLIPBOARD_PREVIEW_WINDOW_LABEL);

    Ok(())
}

/// 平台 hide 收口点；成功后推进生命周期到 `HiddenWarm`，启动空闲销毁计时。
/// 对已隐藏窗口的重复 hide（如主窗口隐藏时的压制路径）也会走到这里，
/// 由生命周期管理器对重复进入 `HiddenWarm` 去重计时。
fn hide_preview_window(app: &AppHandle, window: &WebviewWindow) -> Result<()> {
    #[cfg(target_os = "macos")]
    {
        let _ = window;
        hide_macos_preview_panel(app)?;
    }

    #[cfg(target_os = "windows")]
    window.hide().map_err(|e| anyhow::anyhow!(e))?;

    lifecycle::on_hidden(app, CLIPBOARD_PREVIEW_WINDOW_LABEL, "preview-hide");

    Ok(())
}

#[cfg(target_os = "macos")]
fn ensure_macos_preview_panel(app: &AppHandle, window: &WebviewWindow) -> Result<()> {
    let handle = app.clone();
    let preview_window = window.clone();
    let (tx, rx) = std::sync::mpsc::channel();

    app.run_on_main_thread(move || {
        let result = setup_macos_preview_panel(&handle, &preview_window);
        let _ = tx.send(result);
    })
    .map_err(|e| anyhow::anyhow!(e))?;

    rx.recv()
        .map_err(|e| anyhow::anyhow!("preview panel setup channel closed: {e}"))?
}

#[cfg(target_os = "macos")]
fn setup_macos_preview_panel(app: &AppHandle, window: &WebviewWindow) -> Result<()> {
    let panel = match app.get_webview_panel(CLIPBOARD_PREVIEW_WINDOW_LABEL) {
        Ok(panel) => panel,
        Err(_) => window
            .to_panel::<PreviewPanel>()
            .map_err(|e| anyhow::anyhow!("to_panel failed: {e:?}"))?,
    };

    panel.set_level(PanelLevel::Status.value());
    panel.set_ignores_mouse_events(true);

    Ok(())
}

#[cfg(target_os = "macos")]
fn set_macos_preview_panel_level(app: &AppHandle) -> Result<()> {
    let handle = app.clone();

    app.run_on_main_thread(move || {
        if let Ok(panel) = handle.get_webview_panel(CLIPBOARD_PREVIEW_WINDOW_LABEL) {
            panel.set_level(PanelLevel::Status.value());
            panel.set_ignores_mouse_events(true);
        }
    })
    .map_err(|e| anyhow::anyhow!(e))?;

    Ok(())
}

#[cfg(target_os = "macos")]
fn show_macos_preview_panel(app: &AppHandle, work_area: &PhysicalRect<i32, u32>) -> Result<()> {
    let handle = app.clone();
    let preview_window = get_window(app, CLIPBOARD_PREVIEW_WINDOW_LABEL)?;
    let work_area = *work_area;
    let (tx, rx) = std::sync::mpsc::channel();

    app.run_on_main_thread(move || {
        let result = (|| -> Result<()> {
            apply_preview_window_bounds(&preview_window, &work_area)?;
            let panel = handle
                .get_webview_panel(CLIPBOARD_PREVIEW_WINDOW_LABEL)
                .map_err(|e| anyhow::anyhow!("preview panel not found: {e:?}"))?;
            panel.set_ignores_mouse_events(true);
            panel.set_level(PanelLevel::Status.value());
            panel.show();
            Ok(())
        })();
        let _ = tx.send(result);
    })
    .map_err(|e| anyhow::anyhow!(e))?;

    rx.recv()
        .map_err(|e| anyhow::anyhow!("preview panel show channel closed: {e}"))?
}

#[cfg(target_os = "macos")]
fn hide_macos_preview_panel(app: &AppHandle) -> Result<()> {
    let handle = app.clone();

    app.run_on_main_thread(move || {
        if let Ok(panel) = handle.get_webview_panel(CLIPBOARD_PREVIEW_WINDOW_LABEL) {
            panel.hide();
        }
    })
    .map_err(|e| anyhow::anyhow!(e))?;

    Ok(())
}

/// 将预览窗口重新压到 Windows topmost 栈顶，避免被同为 always-on-top 的主窗口盖住。
#[cfg(target_os = "windows")]
fn raise_windows_preview_window(window: &WebviewWindow, show: bool) -> Result<()> {
    let raw_hwnd = window.hwnd().map_err(|e| anyhow::anyhow!(e))?;
    let hwnd = HWND(raw_hwnd.0 as isize);
    let mut flags = SWP_NOACTIVATE | SWP_NOMOVE | SWP_NOSIZE;

    if show {
        flags |= SWP_SHOWWINDOW;
    }

    unsafe {
        SetWindowPos(hwnd, HWND_TOPMOST, 0, 0, 0, 0, flags).map_err(|e| anyhow::anyhow!(e))?;
    }

    Ok(())
}

fn resolve_preview_monitor(app: &AppHandle) -> Result<tauri::Monitor> {
    let main_window = get_window(app, MAIN_WINDOW_LABEL)?;
    if let Some(monitor) = main_window
        .current_monitor()
        .map_err(|e| anyhow::anyhow!(e))?
    {
        return Ok(monitor);
    }

    main_window
        .primary_monitor()
        .map_err(|e| anyhow::anyhow!(e))?
        .ok_or_else(|| anyhow::anyhow!("primary monitor not found").into())
}

/// 用完整显示器区域作为预览 overlay 边界，避免 macOS Dock 压缩 `work_area` 后截断连线。
fn preview_overlay_bounds(monitor: &tauri::Monitor) -> PhysicalRect<i32, u32> {
    PhysicalRect {
        position: *monitor.position(),
        size: *monitor.size(),
    }
}

fn apply_preview_window_bounds(
    window: &WebviewWindow,
    work_area: &PhysicalRect<i32, u32>,
) -> Result<()> {
    let size = preview_window_size(work_area);

    window
        .set_position(PhysicalPosition::new(
            work_area.position.x,
            work_area.position.y,
        ))
        .map_err(|e| anyhow::anyhow!(e))?;
    window.set_size(size).map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

/// 返回实际预览窗口尺寸；Windows 避免精确全屏触发系统勿扰模式。
fn preview_window_size(work_area: &PhysicalRect<i32, u32>) -> PhysicalSize<u32> {
    #[cfg(target_os = "windows")]
    {
        return PhysicalSize::new(
            work_area.size.width.saturating_sub(1),
            work_area.size.height,
        );
    }

    #[cfg(target_os = "macos")]
    {
        PhysicalSize::new(work_area.size.width, work_area.size.height)
    }
}

fn build_preview_layout(
    anchor: &PreviewAnchorRect,
    scale_factor: f64,
    work_area: &PhysicalRect<i32, u32>,
    main_window: Option<&PreviewMainWindowRect>,
) -> PreviewLayout {
    let overlay_rect = PreviewRect {
        left: 0.0,
        top: 0.0,
        width: work_area.size.width as f64 / scale_factor,
        height: work_area.size.height as f64 / scale_factor,
    };
    let source_rect =
        resolve_source_rect(anchor, scale_factor, work_area, main_window, overlay_rect);
    let (panel_rect, placement) = resolve_panel_rect(source_rect, overlay_rect);

    PreviewLayout {
        overlay_rect,
        source_rect,
        panel_rect,
        placement,
    }
}

fn resolve_source_rect(
    anchor: &PreviewAnchorRect,
    scale_factor: f64,
    work_area: &PhysicalRect<i32, u32>,
    main_window: Option<&PreviewMainWindowRect>,
    overlay_rect: PreviewRect,
) -> PreviewRect {
    let source = if let Some(main_window) = main_window {
        let main_rect = PreviewRect {
            left: (main_window.x - work_area.position.x) as f64 / scale_factor,
            top: (main_window.y - work_area.position.y) as f64 / scale_factor,
            width: main_window.width as f64 / scale_factor,
            height: main_window.height as f64 / scale_factor,
        };
        let source = PreviewRect {
            left: main_rect.left + anchor.left,
            top: main_rect.top + anchor.top,
            width: anchor.width,
            height: anchor.height,
        };

        let pointer_y = anchor.pointer_y.map(|y| main_rect.top + y);

        resolve_pointer_anchor_rect(source, pointer_y)
    } else {
        let source = PreviewRect {
            left: anchor.left,
            top: anchor.top,
            width: anchor.width,
            height: anchor.height,
        };

        resolve_pointer_anchor_rect(source, anchor.pointer_y)
    };

    intersect_rect(source, overlay_rect).unwrap_or_else(|| clamp_rect(source, overlay_rect))
}

fn resolve_pointer_anchor_rect(source: PreviewRect, pointer_y: Option<f64>) -> PreviewRect {
    let Some(pointer_y) = pointer_y else {
        return source;
    };
    let center_y = pointer_y.clamp(source.top, source.bottom());
    let top = center_y - PREVIEW_POINTER_ANCHOR_SIZE / 2.0;

    PreviewRect {
        left: source.left,
        top,
        width: source.width,
        height: PREVIEW_POINTER_ANCHOR_SIZE,
    }
}

fn resolve_panel_rect(
    source_rect: PreviewRect,
    overlay_rect: PreviewRect,
) -> (PreviewRect, PreviewPlacement) {
    let panel_size = (PREVIEW_PANEL_WIDTH, PREVIEW_PANEL_HEIGHT);
    let candidates = [
        (
            PreviewPlacement::Right,
            source_rect.right() + PREVIEW_PANEL_GAP + PREVIEW_PANEL_WIDTH + PREVIEW_PANEL_MARGIN
                <= overlay_rect.right(),
        ),
        (
            PreviewPlacement::Left,
            source_rect.left - PREVIEW_PANEL_GAP - PREVIEW_PANEL_WIDTH - PREVIEW_PANEL_MARGIN
                >= overlay_rect.left,
        ),
        (
            PreviewPlacement::Bottom,
            source_rect.bottom() + PREVIEW_PANEL_GAP + PREVIEW_PANEL_HEIGHT + PREVIEW_PANEL_MARGIN
                <= overlay_rect.bottom(),
        ),
        (
            PreviewPlacement::Top,
            source_rect.top - PREVIEW_PANEL_GAP - PREVIEW_PANEL_HEIGHT - PREVIEW_PANEL_MARGIN
                >= overlay_rect.top,
        ),
    ];

    let placement = candidates
        .iter()
        .find_map(|(placement, fits)| fits.then_some(*placement))
        .unwrap_or(PreviewPlacement::Right);
    let raw = raw_panel_rect(source_rect, placement, panel_size);

    (
        clamp_rect(raw, inset_rect(overlay_rect, PREVIEW_PANEL_MARGIN)),
        placement,
    )
}

fn raw_panel_rect(
    source_rect: PreviewRect,
    placement: PreviewPlacement,
    (width, height): (f64, f64),
) -> PreviewRect {
    let centered_top = source_rect.center_y() - height / 2.0;
    let centered_left = source_rect.center_x() - width / 2.0;

    match placement {
        PreviewPlacement::Right => PreviewRect {
            left: source_rect.right() + PREVIEW_PANEL_GAP,
            top: centered_top,
            width,
            height,
        },
        PreviewPlacement::Left => PreviewRect {
            left: source_rect.left - PREVIEW_PANEL_GAP - width,
            top: centered_top,
            width,
            height,
        },
        PreviewPlacement::Bottom => PreviewRect {
            left: centered_left,
            top: source_rect.bottom() + PREVIEW_PANEL_GAP,
            width,
            height,
        },
        PreviewPlacement::Top => PreviewRect {
            left: centered_left,
            top: source_rect.top - PREVIEW_PANEL_GAP - height,
            width,
            height,
        },
    }
}

fn clamp_rect(rect: PreviewRect, bounds: PreviewRect) -> PreviewRect {
    let max_left = (bounds.right() - rect.width).max(bounds.left);
    let max_top = (bounds.bottom() - rect.height).max(bounds.top);

    PreviewRect {
        left: rect.left.clamp(bounds.left, max_left),
        top: rect.top.clamp(bounds.top, max_top),
        width: rect.width,
        height: rect.height,
    }
}

fn intersect_rect(a: PreviewRect, b: PreviewRect) -> Option<PreviewRect> {
    let left = a.left.max(b.left);
    let top = a.top.max(b.top);
    let right = a.right().min(b.right());
    let bottom = a.bottom().min(b.bottom());

    if right <= left || bottom <= top {
        return None;
    }

    Some(PreviewRect {
        left,
        top,
        width: right - left,
        height: bottom - top,
    })
}

fn inset_rect(rect: PreviewRect, amount: f64) -> PreviewRect {
    PreviewRect {
        left: rect.left + amount,
        top: rect.top + amount,
        width: (rect.width - amount * 2.0).max(1.0),
        height: (rect.height - amount * 2.0).max(1.0),
    }
}

/// 返回主窗口内容区的屏幕几何，用于映射 WebView DOM rect 到预览 overlay 坐标。
fn main_window_rect(app: &AppHandle) -> Option<PreviewMainWindowRect> {
    let window = app.get_webview_window(MAIN_WINDOW_LABEL)?;
    let pos = window.inner_position().ok()?;
    let size = window.inner_size().ok()?;

    Some(PreviewMainWindowRect {
        x: pos.x,
        y: pos.y,
        width: size.width,
        height: size.height,
    })
}

impl PreviewRect {
    fn right(self) -> f64 {
        self.left + self.width
    }

    fn bottom(self) -> f64 {
        self.top + self.height
    }

    fn center_x(self) -> f64 {
        self.left + self.width / 2.0
    }

    fn center_y(self) -> f64 {
        self.top + self.height / 2.0
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn overlay() -> PreviewRect {
        PreviewRect {
            left: 0.0,
            top: 0.0,
            width: 1200.0,
            height: 800.0,
        }
    }

    #[test]
    fn places_panel_on_right_when_space_exists() {
        let source = PreviewRect {
            left: 240.0,
            top: 200.0,
            width: 120.0,
            height: 40.0,
        };
        let (panel, placement) = resolve_panel_rect(source, overlay());

        assert!(matches!(placement, PreviewPlacement::Right));
        assert!(panel.left > source.right());
    }

    #[test]
    fn places_panel_on_left_when_right_side_is_tight() {
        let source = PreviewRect {
            left: 980.0,
            top: 200.0,
            width: 120.0,
            height: 40.0,
        };
        let (panel, placement) = resolve_panel_rect(source, overlay());

        assert!(matches!(placement, PreviewPlacement::Left));
        assert!(panel.right() < source.left);
    }

    #[test]
    fn clamps_panel_inside_overlay_margin() {
        let source = PreviewRect {
            left: 580.0,
            top: 740.0,
            width: 80.0,
            height: 40.0,
        };
        let (panel, _) = resolve_panel_rect(source, overlay());

        assert!(panel.left >= PREVIEW_PANEL_MARGIN);
        assert!(panel.top >= PREVIEW_PANEL_MARGIN);
        assert!(panel.right() <= overlay().right() - PREVIEW_PANEL_MARGIN);
        assert!(panel.bottom() <= overlay().bottom() - PREVIEW_PANEL_MARGIN);
    }

    #[test]
    fn maps_anchor_from_main_window_to_overlay_local_rect() {
        let work_area = PhysicalRect {
            position: PhysicalPosition::new(100, 50),
            size: PhysicalSize::new(2400, 1600),
        };
        let main = PreviewMainWindowRect {
            x: 300,
            y: 250,
            width: 800,
            height: 600,
        };
        let anchor = PreviewAnchorRect {
            left: 20.0,
            pointer_y: None,
            top: 30.0,
            width: 100.0,
            height: 40.0,
        };
        let source = resolve_source_rect(
            &anchor,
            2.0,
            &work_area,
            Some(&main),
            PreviewRect {
                left: 0.0,
                top: 0.0,
                width: 1200.0,
                height: 800.0,
            },
        );

        assert_eq!(source.left, 120.0);
        assert_eq!(source.top, 130.0);
        assert_eq!(source.width, 100.0);
        assert_eq!(source.height, 40.0);
    }

    #[test]
    fn keeps_source_rect_on_clipboard_card_when_card_touches_main_edge() {
        let work_area = PhysicalRect {
            position: PhysicalPosition::new(100, 50),
            size: PhysicalSize::new(2400, 1600),
        };
        let main = PreviewMainWindowRect {
            x: 300,
            y: 250,
            width: 800,
            height: 600,
        };
        let anchor = PreviewAnchorRect {
            left: -8.0,
            pointer_y: None,
            top: 40.0,
            width: 120.0,
            height: 80.0,
        };
        let source = resolve_source_rect(
            &anchor,
            2.0,
            &work_area,
            Some(&main),
            PreviewRect {
                left: 0.0,
                top: 0.0,
                width: 1200.0,
                height: 800.0,
            },
        );

        assert_eq!(source.left, 92.0);
        assert_eq!(source.width, 120.0);
    }

    #[test]
    fn follows_pointer_y_inside_clipboard_card() {
        let work_area = PhysicalRect {
            position: PhysicalPosition::new(100, 50),
            size: PhysicalSize::new(2400, 1600),
        };
        let main = PreviewMainWindowRect {
            x: 300,
            y: 250,
            width: 800,
            height: 600,
        };
        let anchor = PreviewAnchorRect {
            left: 20.0,
            pointer_y: Some(92.0),
            top: 40.0,
            width: 120.0,
            height: 100.0,
        };
        let source = resolve_source_rect(
            &anchor,
            2.0,
            &work_area,
            Some(&main),
            PreviewRect {
                left: 0.0,
                top: 0.0,
                width: 1200.0,
                height: 800.0,
            },
        );

        assert_eq!(source.left, 120.0);
        assert_eq!(source.top, 191.5);
        assert_eq!(source.width, 120.0);
        assert_eq!(source.height, PREVIEW_POINTER_ANCHOR_SIZE);
    }
}
