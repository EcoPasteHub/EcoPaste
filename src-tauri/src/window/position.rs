use tauri::{PhysicalPosition, PhysicalSize, WebviewWindow};

use crate::core::Result;
use crate::settings::WindowPosition;

#[derive(Clone, Copy, Debug, PartialEq)]
struct MonitorInfo {
    position: PhysicalPosition<i32>,
    size: PhysicalSize<u32>,
    scale_factor: f64,
}

impl From<&tauri::Monitor> for MonitorInfo {
    fn from(monitor: &tauri::Monitor) -> Self {
        Self {
            position: *monitor.position(),
            size: *monitor.size(),
            scale_factor: monitor.scale_factor(),
        }
    }
}

#[cfg(any(target_os = "windows", test))]
impl MonitorInfo {
    fn contains(&self, x: f64, y: f64) -> bool {
        let left = self.position.x as f64;
        let top = self.position.y as f64;

        x >= left
            && x < left + self.size.width as f64
            && y >= top
            && y < top + self.size.height as f64
    }
}

/// 在 Windows 显示器列表中找出光标所在的显示器。
///
/// Windows 虚拟桌面坐标统一是物理像素，`GetCursorPos` 与 `rcMonitor` 同处这一空间，
/// 不能按某块屏的缩放比例换算，否则缩放 > 100% 时副屏坐标会被压回主屏范围。
#[cfg(any(target_os = "windows", test))]
fn monitor_at_cursor(
    cursor: PhysicalPosition<f64>,
    monitors: &[MonitorInfo],
) -> Option<MonitorInfo> {
    monitors
        .iter()
        .copied()
        .find(|monitor| monitor.contains(cursor.x, cursor.y))
}

fn monitor_from_cursor(
    window: &WebviewWindow,
) -> Result<Option<(MonitorInfo, PhysicalPosition<f64>)>> {
    let cursor = window.cursor_position().map_err(|e| anyhow::anyhow!(e))?;

    #[cfg(target_os = "windows")]
    let monitor = {
        let monitors: Vec<MonitorInfo> = window
            .available_monitors()
            .map_err(|e| anyhow::anyhow!(e))?
            .iter()
            .map(MonitorInfo::from)
            .collect();

        monitor_at_cursor(cursor, &monitors)
    };

    #[cfg(target_os = "macos")]
    let monitor = {
        let scale = window.scale_factor().map_err(|e| anyhow::anyhow!(e))?;
        let logical = cursor.to_logical::<f64>(scale);

        window
            .monitor_from_point(logical.x, logical.y)
            .map_err(|e| anyhow::anyhow!(e))?
            .as_ref()
            .map(MonitorInfo::from)
    };

    Ok(monitor.map(|monitor| (monitor, cursor)))
}

pub fn position_window(window: &WebviewWindow, position: WindowPosition) -> Result<()> {
    let Some((monitor, cursor)) = monitor_from_cursor(window)? else {
        return Ok(());
    };

    match position {
        WindowPosition::Remember => {}
        WindowPosition::FollowCursor => apply_follow(window, &monitor, &cursor)?,
        WindowPosition::Center => apply_center(window, &monitor)?,
    }

    Ok(())
}

fn apply_follow(
    window: &WebviewWindow,
    monitor: &MonitorInfo,
    cursor: &PhysicalPosition<f64>,
) -> Result<()> {
    let win_size = window.inner_size().map_err(|e| anyhow::anyhow!(e))?;
    let win_scale = window.scale_factor().map_err(|e| anyhow::anyhow!(e))?;

    window
        .set_position(follow_position(*cursor, monitor, win_size, win_scale))
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

/// 窗口落到 `monitor` 后的物理尺寸。
///
/// 跨缩放比例移动时系统会保持逻辑尺寸、按目标屏缩放重算物理尺寸，
/// 所以定位必须用目标屏上的尺寸，而不是移动前的 `inner_size`。
fn size_on_monitor(
    win_size: PhysicalSize<u32>,
    win_scale: f64,
    monitor: &MonitorInfo,
) -> PhysicalSize<f64> {
    win_size
        .to_logical::<f64>(win_scale)
        .to_physical(monitor.scale_factor)
}

fn follow_position(
    cursor: PhysicalPosition<f64>,
    monitor: &MonitorInfo,
    win_size: PhysicalSize<u32>,
    win_scale: f64,
) -> PhysicalPosition<i32> {
    let size = size_on_monitor(win_size, win_scale, monitor);
    let mon_x = monitor.position.x as f64;
    let mon_y = monitor.position.y as f64;
    let mon_w = monitor.size.width as f64;
    let mon_h = monitor.size.height as f64;

    let x = cursor.x.min(mon_x + mon_w - size.width);
    let y = cursor.y.min(mon_y + mon_h - size.height);

    PhysicalPosition::new(x.round() as i32, y.round() as i32)
}

/// 将窗口居中到当前光标所在显示器。
/// 用于存档位置已失效（显示器被拔出）时的 fallback。
pub(super) fn center_on_cursor_monitor(window: &WebviewWindow) -> Result<()> {
    let Some((monitor, _)) = monitor_from_cursor(window)? else {
        return Ok(());
    };
    apply_center(window, &monitor)
}

fn apply_center(window: &WebviewWindow, monitor: &MonitorInfo) -> Result<()> {
    let win_size = window.inner_size().map_err(|e| anyhow::anyhow!(e))?;
    let win_scale = window.scale_factor().map_err(|e| anyhow::anyhow!(e))?;

    window
        .set_position(center_position(monitor, win_size, win_scale))
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}

fn center_position(
    monitor: &MonitorInfo,
    win_size: PhysicalSize<u32>,
    win_scale: f64,
) -> PhysicalPosition<i32> {
    let size = size_on_monitor(win_size, win_scale, monitor);
    let mon_x = monitor.position.x as f64;
    let mon_y = monitor.position.y as f64;
    let mon_w = monitor.size.width as f64;
    let mon_h = monitor.size.height as f64;

    let x = mon_x + (mon_w - size.width) / 2.0;
    let y = mon_y + (mon_h - size.height) / 2.0;

    PhysicalPosition::new(x.round() as i32, y.round() as i32)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn monitor(x: i32, y: i32, width: u32, height: u32) -> MonitorInfo {
        scaled_monitor(x, y, width, height, 1.0)
    }

    fn scaled_monitor(x: i32, y: i32, width: u32, height: u32, scale_factor: f64) -> MonitorInfo {
        MonitorInfo {
            position: PhysicalPosition::new(x, y),
            size: PhysicalSize::new(width, height),
            scale_factor,
        }
    }

    #[test]
    fn follow_position_keeps_cursor_when_window_fits() {
        let primary = monitor(0, 0, 1920, 1080);
        let cursor = PhysicalPosition::new(500.0, 300.0);

        let position = follow_position(cursor, &primary, PhysicalSize::new(360, 600), 1.0);

        assert_eq!(position, PhysicalPosition::new(500, 300));
    }

    #[test]
    fn follow_position_clamps_with_window_size_on_higher_scale_monitor() {
        let secondary = scaled_monitor(1920, 0, 3840, 2160, 2.0);
        let cursor = PhysicalPosition::new(5700.0, 2100.0);

        let position = follow_position(cursor, &secondary, PhysicalSize::new(360, 600), 1.0);

        assert_eq!(
            position,
            PhysicalPosition::new(1920 + 3840 - 720, 2160 - 1200)
        );
    }

    #[test]
    fn follow_position_clamps_with_window_size_on_lower_scale_monitor() {
        let secondary = scaled_monitor(3840, 0, 1920, 1080, 1.0);
        let cursor = PhysicalPosition::new(5700.0, 1000.0);

        let position = follow_position(cursor, &secondary, PhysicalSize::new(720, 1200), 2.0);

        assert_eq!(
            position,
            PhysicalPosition::new(3840 + 1920 - 360, 1080 - 600)
        );
    }

    #[test]
    fn center_position_uses_window_size_on_target_monitor() {
        let secondary = scaled_monitor(1920, 0, 3840, 2160, 2.0);

        let position = center_position(&secondary, PhysicalSize::new(360, 600), 1.0);

        assert_eq!(
            position,
            PhysicalPosition::new(1920 + (3840 - 720) / 2, (2160 - 1200) / 2)
        );
    }

    #[test]
    fn monitor_at_cursor_picks_secondary_when_primary_is_scaled() {
        let primary = monitor(0, 0, 3840, 2160);
        let secondary = monitor(3840, 0, 1920, 1080);
        let cursor = PhysicalPosition::new(4800.0, 540.0);

        let found = monitor_at_cursor(cursor, &[primary, secondary]);

        assert_eq!(found, Some(secondary));
    }

    #[test]
    fn monitor_at_cursor_picks_primary_when_cursor_on_primary() {
        let primary = monitor(0, 0, 3840, 2160);
        let secondary = monitor(-1920, 0, 1920, 1080);
        let cursor = PhysicalPosition::new(3000.0, 2000.0);

        let found = monitor_at_cursor(cursor, &[primary, secondary]);

        assert_eq!(found, Some(primary));
    }

    #[test]
    fn monitor_at_cursor_returns_none_outside_all_monitors() {
        let primary = monitor(0, 0, 1920, 1080);
        let cursor = PhysicalPosition::new(5000.0, 5000.0);

        assert_eq!(monitor_at_cursor(cursor, &[primary]), None);
    }
}
