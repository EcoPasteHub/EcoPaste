use serde::{Deserialize, Serialize};
use tauri::{PhysicalPosition, PhysicalSize, WebviewWindow};

use crate::core::Result;

#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum WindowPosition {
    Remember,
    Follow,
    Center,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum WindowStyle {
    Standard,
    Dock,
}

const DOCK_HEIGHT: u32 = 400;

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

pub fn position_window(
    window: &WebviewWindow,
    style: WindowStyle,
    position: WindowPosition,
) -> Result<()> {
    let Some((monitor, cursor)) = monitor_from_cursor(window)? else {
        return Ok(());
    };

    match style {
        WindowStyle::Dock => apply_dock(window, &monitor)?,
        WindowStyle::Standard => match position {
            WindowPosition::Remember => {}
            WindowPosition::Follow => apply_follow(window, &monitor, &cursor)?,
            WindowPosition::Center => apply_center(window, &monitor)?,
        },
    }

    Ok(())
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

fn apply_dock(window: &WebviewWindow, monitor: &MonitorInfo) -> Result<()> {
    let mon_x = monitor.position.x;
    let mon_y = monitor.position.y;
    let mon_w = monitor.size.width;
    let mon_h = monitor.size.height;

    let y = mon_y + mon_h as i32 - DOCK_HEIGHT as i32;

    window
        .set_size(PhysicalSize::new(mon_w, DOCK_HEIGHT))
        .map_err(|e| anyhow::anyhow!(e))?;
    window
        .set_position(PhysicalPosition::new(mon_x, y))
        .map_err(|e| anyhow::anyhow!(e))?;
    Ok(())
}
