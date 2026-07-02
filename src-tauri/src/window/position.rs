use tauri::{PhysicalPosition, PhysicalSize, WebviewWindow};

use crate::core::Result;
use crate::settings::WindowPosition;

const SHEET_LOGICAL_WIDTH: f64 = 480.0;
const FLOATING_SHEET_LOGICAL_HEIGHT: f64 = 576.0;
const BOTTOM_SHEET_LOGICAL_HEIGHT: f64 = 360.0;

struct MonitorInfo {
    position: PhysicalPosition<i32>,
    size: PhysicalSize<u32>,
}

fn monitor_from_cursor(
    window: &WebviewWindow,
) -> Result<Option<(MonitorInfo, PhysicalPosition<f64>)>> {
    let cursor = window.cursor_position().map_err(|e| anyhow::anyhow!(e))?;
    let scale = window.scale_factor().map_err(|e| anyhow::anyhow!(e))?;

    let logical = cursor.to_logical::<f64>(scale);

    let monitor = window
        .monitor_from_point(logical.x, logical.y)
        .map_err(|e| anyhow::anyhow!(e))?;

    let Some(monitor) = monitor else {
        return Ok(None);
    };

    Ok(Some((
        MonitorInfo {
            position: *monitor.position(),
            size: *monitor.size(),
        },
        cursor,
    )))
}

pub fn position_window(window: &WebviewWindow, position: WindowPosition) -> Result<()> {
    let Some((monitor, cursor)) = monitor_from_cursor(window)? else {
        return Ok(());
    };

    let scale = window.scale_factor().map_err(|e| anyhow::anyhow!(e))?;

    match position {
        WindowPosition::Remember => {
            apply_floating_sheet_size(window, &monitor, scale)?;
        }
        WindowPosition::FollowCursor => apply_follow(window, &monitor, &cursor, scale)?,
        WindowPosition::Center => apply_center(window, &monitor, scale)?,
        WindowPosition::BottomSheet => apply_bottom_sheet(window, &monitor, scale)?,
    }

    Ok(())
}

fn apply_follow(
    window: &WebviewWindow,
    monitor: &MonitorInfo,
    cursor: &PhysicalPosition<f64>,
    scale: f64,
) -> Result<()> {
    let win_size = apply_floating_sheet_size(window, monitor, scale)?;
    let mon_x = monitor.position.x as f64;
    let mon_y = monitor.position.y as f64;
    let mon_w = monitor.size.width as f64;
    let mon_h = monitor.size.height as f64;

    let x = cursor.x.min(mon_x + mon_w - win_size.width as f64);
    let y = cursor.y.min(mon_y + mon_h - win_size.height as f64);

    window
        .set_position(PhysicalPosition::new(x.round() as i32, y.round() as i32))
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

/// 将窗口居中到当前光标所在显示器。
/// 用于存档位置已失效（显示器被拔出）时的 fallback。
pub(super) fn center_on_cursor_monitor(window: &WebviewWindow) -> Result<()> {
    let Some((monitor, _)) = monitor_from_cursor(window)? else {
        return Ok(());
    };
    let scale = window.scale_factor().map_err(|e| anyhow::anyhow!(e))?;

    apply_center(window, &monitor, scale)
}

fn apply_center(window: &WebviewWindow, monitor: &MonitorInfo, scale: f64) -> Result<()> {
    let win_size = apply_floating_sheet_size(window, monitor, scale)?;
    let mon_x = monitor.position.x as f64;
    let mon_y = monitor.position.y as f64;
    let mon_w = monitor.size.width as f64;
    let mon_h = monitor.size.height as f64;

    let x = mon_x + (mon_w - win_size.width as f64) / 2.0;
    let y = mon_y + (mon_h - win_size.height as f64) / 2.0;

    window
        .set_position(PhysicalPosition::new(x.round() as i32, y.round() as i32))
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

fn apply_bottom_sheet(window: &WebviewWindow, monitor: &MonitorInfo, scale: f64) -> Result<()> {
    let size = sheet_size(monitor, scale, true);
    let x = monitor.position.x;
    let y = monitor.position.y + monitor.size.height.saturating_sub(size.height) as i32;

    window.set_size(size).map_err(|e| anyhow::anyhow!(e))?;
    window
        .set_position(PhysicalPosition::new(x, y))
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

fn apply_floating_sheet_size(
    window: &WebviewWindow,
    monitor: &MonitorInfo,
    scale: f64,
) -> Result<PhysicalSize<u32>> {
    let size = sheet_size(monitor, scale, false);

    window.set_size(size).map_err(|e| anyhow::anyhow!(e))?;

    Ok(size)
}

fn sheet_size(monitor: &MonitorInfo, scale: f64, full_width: bool) -> PhysicalSize<u32> {
    let preferred_width = if full_width {
        monitor.size.width
    } else {
        (SHEET_LOGICAL_WIDTH * scale).round() as u32
    };
    let logical_height = if full_width {
        BOTTOM_SHEET_LOGICAL_HEIGHT
    } else {
        FLOATING_SHEET_LOGICAL_HEIGHT
    };
    let preferred_height = (logical_height * scale).round() as u32;

    PhysicalSize::new(
        preferred_width.min(monitor.size.width),
        preferred_height.min(monitor.size.height),
    )
}
