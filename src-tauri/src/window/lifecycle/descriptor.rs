//! 窗口声明式描述（descriptor）与静态 registry。
//!
//! 每个窗口只在这里声明一次；show / hide / close / 销毁重建等主流程统一查 registry，
//! 不再在各处散写 `if label == ...` 分支。新增窗口时在 [`DESCRIPTORS`] 加一条即可获得
//! 生命周期能力。

use tauri::AppHandle;

use super::super::{
    build_preference_window, CLIPBOARD_PREVIEW_WINDOW_LABEL, MAIN_WINDOW_LABEL,
    PREFERENCE_WINDOW_LABEL,
};
use crate::core::Result;

#[cfg(target_os = "windows")]
use crate::menu::context_window::{CONTEXT_MENU_WINDOW_LABEL, CONTEXT_SUBMENU_WINDOW_LABEL};

/// 窗口保留 / 销毁策略。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RetainPolicy {
    /// 永久保留实例，隐藏后不销毁（main / preview / context-menu）。
    Permanent,
    /// 隐藏空闲超过 [`IDLE_DESTROY_SECS`](super::IDLE_DESTROY_SECS) 后销毁 WebView，打开时重建。
    DestroyWhenIdle,
}

/// 单个窗口的声明式描述。
#[derive(Clone, Copy)]
pub struct WindowDescriptor {
    /// Tauri window label。
    pub label: &'static str,
    /// 是否向前端广播生命周期事件（`window://lifecycle`）。
    pub emits_lifecycle: bool,
    /// 保留 / 销毁策略。
    pub retain_policy: RetainPolicy,
    /// 按需重建函数。`DestroyWhenIdle` 窗口被销毁后，`show_window` 用它重新建窗；
    /// `Permanent` 窗口由 Tauri 配置预创建，无需重建，为 `None`。
    pub build: Option<fn(&AppHandle) -> Result<()>>,
}

/// 全部窗口的静态声明表。新增窗口在此追加。
static DESCRIPTORS: &[WindowDescriptor] = &[
    WindowDescriptor {
        label: MAIN_WINDOW_LABEL,
        emits_lifecycle: true,
        retain_policy: RetainPolicy::Permanent,
        build: None,
    },
    WindowDescriptor {
        label: PREFERENCE_WINDOW_LABEL,
        emits_lifecycle: true,
        retain_policy: RetainPolicy::DestroyWhenIdle,
        build: Some(build_preference_window),
    },
    WindowDescriptor {
        label: CLIPBOARD_PREVIEW_WINDOW_LABEL,
        emits_lifecycle: true,
        retain_policy: RetainPolicy::Permanent,
        build: None,
    },
    #[cfg(target_os = "windows")]
    WindowDescriptor {
        label: CONTEXT_MENU_WINDOW_LABEL,
        emits_lifecycle: true,
        retain_policy: RetainPolicy::Permanent,
        build: None,
    },
    #[cfg(target_os = "windows")]
    WindowDescriptor {
        label: CONTEXT_SUBMENU_WINDOW_LABEL,
        emits_lifecycle: true,
        retain_policy: RetainPolicy::Permanent,
        build: None,
    },
];

/// 按 label 查 descriptor；未登记的 label 返回 `None`。
pub fn descriptor_for(label: &str) -> Option<&'static WindowDescriptor> {
    DESCRIPTORS.iter().find(|d| d.label == label)
}
