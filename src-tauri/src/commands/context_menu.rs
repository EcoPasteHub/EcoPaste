//! Command wrappers for the Windows custom context-menu webviews.

use serde::{Deserialize, Serialize};
use tauri::AppHandle;

use crate::core::Result;
use crate::menu::clipboard_item::ClipboardMenuAction;

/// Anchor rectangle relative to the root menu webview viewport.
#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ContextSubmenuAnchor {
    pub left: f64,
    pub top: f64,
    pub width: f64,
    pub height: f64,
}

/// Single group row rendered by the submenu window.
#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ContextSubmenuGroupInput {
    pub checked: bool,
    pub id: String,
    pub label: String,
}

/// Full payload needed to show the submenu window.
#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ShowContextSubmenuInput {
    pub action: ClipboardMenuAction,
    pub anchor: ContextSubmenuAnchor,
    pub groups: Vec<ContextSubmenuGroupInput>,
    pub item_id: String,
}

/// Shows the secondary Windows custom context-menu window.
#[tauri::command]
pub async fn show_context_submenu(app: AppHandle, input: ShowContextSubmenuInput) -> Result<()> {
    #[cfg(target_os = "windows")]
    {
        return crate::menu::context_window::show_submenu(&app, input);
    }

    #[cfg(not(target_os = "windows"))]
    {
        let _ = app;
        let _ = input;

        Ok(())
    }
}

/// Hides the secondary Windows custom context-menu window.
#[tauri::command]
pub async fn hide_context_submenu(app: AppHandle) -> Result<()> {
    #[cfg(target_os = "windows")]
    {
        crate::menu::context_window::hide_submenu(&app);
    }

    #[cfg(not(target_os = "windows"))]
    {
        let _ = app;
    }

    Ok(())
}

/// Hides both Windows custom context-menu windows.
#[tauri::command]
pub async fn hide_context_menus(app: AppHandle) -> Result<()> {
    #[cfg(target_os = "windows")]
    {
        crate::menu::context_window::hide(&app);
    }

    #[cfg(not(target_os = "windows"))]
    {
        let _ = app;
    }

    Ok(())
}
