use tauri::{LogicalSize, Monitor, PhysicalPosition, PhysicalSize, Size, WebviewWindow};

use crate::core::Result;
use crate::settings::WindowPosition;

/// Logical minimum size of the classic floating clipboard panel.
pub const CLIPBOARD_PANEL_MIN_WIDTH: f64 = 360.0;
pub const CLIPBOARD_PANEL_MIN_HEIGHT: f64 = 600.0;
/// Logical minimum size of the bottom history shelf.
pub const CLIPBOARD_DOCK_MIN_WIDTH: f64 = 640.0;
pub const CLIPBOARD_DOCK_MIN_HEIGHT_LOGICAL: f64 = 220.0;
pub const CLIPBOARD_DOCK_DEFAULT_HEIGHT_LOGICAL: f64 = 320.0;

const DOCK_SNAP_TOLERANCE_PX: i32 = 2;

struct MonitorInfo {
    position: PhysicalPosition<i32>,
    size: PhysicalSize<u32>,
    work_area_position: PhysicalPosition<i32>,
    work_area_size: PhysicalSize<u32>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct DockRect {
    pub position: PhysicalPosition<i32>,
    pub size: PhysicalSize<u32>,
}

fn monitor_info(monitor: &Monitor) -> MonitorInfo {
    let work_area = monitor.work_area();

    MonitorInfo {
        position: *monitor.position(),
        size: *monitor.size(),
        work_area_position: work_area.position,
        work_area_size: work_area.size,
    }
}

fn resolve_target_monitor(
    window: &WebviewWindow,
) -> Result<Option<(MonitorInfo, Option<PhysicalPosition<f64>>)>> {
    if let Some((monitor, cursor)) = monitor_from_cursor(window)? {
        return Ok(Some((monitor, Some(cursor))));
    }

    let Some(monitor) = window.primary_monitor().map_err(|e| anyhow::anyhow!(e))? else {
        return Ok(None);
    };

    Ok(Some((monitor_info(&monitor), None)))
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

    Ok(Some((monitor_info(&monitor), cursor)))
}

pub fn position_window(window: &WebviewWindow, position: WindowPosition) -> Result<()> {
    let Some((monitor, cursor)) = resolve_target_monitor(window)? else {
        return Ok(());
    };

    match position {
        WindowPosition::Remember => {}
        WindowPosition::FollowCursor => {
            let Some(cursor) = cursor else {
                apply_center(window, &monitor)?;
                return Ok(());
            };
            apply_follow(window, &monitor, &cursor)?;
        }
        WindowPosition::Center => apply_center(window, &monitor)?,
        WindowPosition::Bottom => apply_bottom(window, &monitor, None)?,
    }

    Ok(())
}

/// Dock the clipboard window to the bottom of the current display work area.
pub fn apply_bottom_dock(window: &WebviewWindow, height: Option<u32>) -> Result<()> {
    let Some((monitor, _)) = resolve_target_monitor(window)? else {
        return Ok(());
    };

    apply_bottom(window, &monitor, height)
}

/// Restore classic panel min/max size after leaving dock mode.
pub fn apply_panel_constraints(window: &WebviewWindow) -> Result<()> {
    window
        .set_min_size(Some(Size::Logical(LogicalSize::new(
            CLIPBOARD_PANEL_MIN_WIDTH,
            CLIPBOARD_PANEL_MIN_HEIGHT,
        ))))
        .map_err(|e| anyhow::anyhow!(e))?;
    window
        .set_max_size(None::<Size>)
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

/// Compute a full-work-area-width shelf sitting above the Dock or taskbar.
pub fn compute_dock_rect(
    work_area_position: PhysicalPosition<i32>,
    work_area_size: PhysicalSize<u32>,
    height: u32,
) -> DockRect {
    let max_height = (work_area_size.height / 2).max(1);
    let height = height.clamp(1, max_height).min(work_area_size.height);
    let y = work_area_position.y + work_area_size.height as i32 - height as i32;

    DockRect {
        position: PhysicalPosition::new(work_area_position.x, y),
        size: PhysicalSize::new(work_area_size.width, height),
    }
}

fn physical_from_logical(logical: f64, scale: f64) -> u32 {
    (logical * scale).round().max(1.0) as u32
}

fn same_dock_rect(
    current_pos: PhysicalPosition<i32>,
    current_size: PhysicalSize<u32>,
    target: DockRect,
) -> bool {
    (current_pos.x - target.position.x).abs() <= DOCK_SNAP_TOLERANCE_PX
        && (current_pos.y - target.position.y).abs() <= DOCK_SNAP_TOLERANCE_PX
        && (current_size.width as i32 - target.size.width as i32).abs() <= DOCK_SNAP_TOLERANCE_PX
        && (current_size.height as i32 - target.size.height as i32).abs() <= DOCK_SNAP_TOLERANCE_PX
}

fn apply_follow(
    window: &WebviewWindow,
    monitor: &MonitorInfo,
    cursor: &PhysicalPosition<f64>,
) -> Result<()> {
    let win_size = window.inner_size().map_err(|e| anyhow::anyhow!(e))?;
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
    let Some((monitor, _)) = resolve_target_monitor(window)? else {
        return Ok(());
    };
    apply_center(window, &monitor)
}

fn apply_center(window: &WebviewWindow, monitor: &MonitorInfo) -> Result<()> {
    let win_size = window.inner_size().map_err(|e| anyhow::anyhow!(e))?;
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

fn apply_bottom(window: &WebviewWindow, monitor: &MonitorInfo, height: Option<u32>) -> Result<()> {
    let scale = window.scale_factor().map_err(|e| anyhow::anyhow!(e))?;
    let min_height = physical_from_logical(CLIPBOARD_DOCK_MIN_HEIGHT_LOGICAL, scale);
    let default_height = physical_from_logical(CLIPBOARD_DOCK_DEFAULT_HEIGHT_LOGICAL, scale);
    let max_height = (monitor.work_area_size.height / 2).max(min_height);
    let panel_height = physical_from_logical(CLIPBOARD_PANEL_MIN_HEIGHT, scale);
    let current = window.inner_size().map_err(|e| anyhow::anyhow!(e))?;
    let requested = height
        .unwrap_or_else(|| {
            // The floating panel is ~360x600. Never reuse that frame as a "shelf".
            let looks_like_panel =
                current.width + 80 < monitor.work_area_size.width || current.height >= panel_height;

            if looks_like_panel {
                default_height
            } else {
                current.height
            }
        })
        .clamp(min_height, max_height);
    let target = compute_dock_rect(
        monitor.work_area_position,
        monitor.work_area_size,
        requested,
    );
    let current_pos = window.outer_position().map_err(|e| anyhow::anyhow!(e))?;

    apply_dock_constraints(window, monitor, scale)?;

    if same_dock_rect(current_pos, current, target) {
        return Ok(());
    }

    // Size then position then size again: AppKit can ignore the first resize when
    // the panel is still hidden or when min/max constraints just changed.
    window
        .set_size(target.size)
        .map_err(|e| anyhow::anyhow!(e))?;
    window
        .set_position(target.position)
        .map_err(|e| anyhow::anyhow!(e))?;
    window
        .set_size(target.size)
        .map_err(|e| anyhow::anyhow!(e))?;
    window
        .set_position(target.position)
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

fn apply_dock_constraints(window: &WebviewWindow, monitor: &MonitorInfo, scale: f64) -> Result<()> {
    window
        .set_min_size(Some(Size::Logical(LogicalSize::new(
            CLIPBOARD_DOCK_MIN_WIDTH,
            CLIPBOARD_DOCK_MIN_HEIGHT_LOGICAL,
        ))))
        .map_err(|e| anyhow::anyhow!(e))?;

    let max_height = (monitor.work_area_size.height / 2).max(physical_from_logical(
        CLIPBOARD_DOCK_MIN_HEIGHT_LOGICAL,
        scale,
    ));
    window
        .set_max_size(Some(Size::Physical(PhysicalSize::new(
            monitor.work_area_size.width,
            max_height,
        ))))
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dock_rect_sits_on_the_work_area_bottom() {
        let rect = compute_dock_rect(
            PhysicalPosition::new(100, 50),
            PhysicalSize::new(1440, 900),
            320,
        );

        assert_eq!(rect.position.x, 100);
        assert_eq!(rect.position.y, 50 + 900 - 320);
        assert_eq!(rect.size.width, 1440);
        assert_eq!(rect.size.height, 320);
    }

    #[test]
    fn dock_rect_clamps_height_to_half_the_work_area() {
        let rect = compute_dock_rect(
            PhysicalPosition::new(0, 0),
            PhysicalSize::new(1920, 1080),
            900,
        );

        assert_eq!(rect.size.height, 540);
        assert_eq!(rect.position.y, 1080 - 540);
    }
}
