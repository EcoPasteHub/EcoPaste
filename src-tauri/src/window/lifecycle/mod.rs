//! 统一窗口生命周期管理器。
//!
//! 持有所有窗口的 phase / generation 状态，把 show / hide / toggle / close / 销毁重建
//! 路径收口到这里。在 `window://visibility` 之外广播单一 `window://lifecycle` 事件
//! （带 `phase` 字段），前端据此镜像每个窗口的生命周期阶段。
//!
//! 销毁策略：`DestroyWhenIdle` 窗口（当前为 preference 与 clipboard-preview）隐藏后
//! 启动空闲计时器，超过 [`IDLE_DESTROY_SECS`] 仍隐藏则销毁 WebView 释放资源；再次打开时
//! 经 descriptor 的 `build` 重建（preference 走 `show_window`，preview 走预览模块按需 ensure
//! 建窗）。`generation` 用于让「显示又隐藏」期间的过期计时器自动失效。

mod descriptor;

pub use descriptor::{descriptor_for, RetainPolicy};

use std::collections::HashMap;
use std::sync::Mutex;
use std::thread;
use std::time::Duration;

use tauri::{AppHandle, Emitter, Manager};

/// 非主窗口隐藏空闲超过此时长后销毁 WebView。
pub const IDLE_DESTROY_SECS: u64 = 60;

/// 生命周期阶段。
#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub enum LifecyclePhase {
    /// 窗口已登记但尚未收到任何转换：作为首次转换日志的「来源」哨兵阶段。
    Created,
    /// 前端完成基础初始化，可安全 show / emit payload。
    Ready,
    /// 窗口可见。
    Visible,
    /// 刚隐藏，保留实例，允许快速恢复；`DestroyWhenIdle` 窗口在此阶段计时销毁。
    HiddenWarm,
    /// WebView 已销毁，仅保留 descriptor；再次打开时重建。
    Destroyed,
}

/// 单个窗口的运行时生命周期状态。
struct RuntimeState {
    phase: LifecyclePhase,
    /// 单调递增代次：每次进入 `Visible` 自增。空闲销毁计时器捕获进入 `HiddenWarm` 时的
    /// 代次，到点若代次已变（窗口期间被重新显示过），说明计时器已过期，直接放弃销毁。
    generation: u64,
}

/// 生命周期事件名。与 `window://visibility` 并存：visibility 只表达布尔可见性，
/// lifecycle 表达完整阶段。
const WINDOW_LIFECYCLE_EVENT: &str = "window://lifecycle";

#[derive(Clone, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct LifecyclePayload<'a> {
    label: &'a str,
    phase: LifecyclePhase,
    generation: u64,
    reason: &'a str,
    visible: bool,
}

/// 窗口生命周期管理器，存入 Tauri `State`。
pub struct WindowLifecycleManager {
    states: Mutex<HashMap<String, RuntimeState>>,
}

impl WindowLifecycleManager {
    pub fn new() -> Self {
        Self {
            states: Mutex::new(HashMap::new()),
        }
    }

    fn with_states<R>(&self, f: impl FnOnce(&mut HashMap<String, RuntimeState>) -> R) -> R {
        let mut guard = self.states.lock().unwrap_or_else(|poisoned| {
            log::error!("window lifecycle mutex poisoned, recovering");
            poisoned.into_inner()
        });
        f(&mut guard)
    }

    /// 转移窗口到新阶段：更新状态、广播 `window://lifecycle`，并按需启动空闲销毁计时器。
    ///
    /// `reason` 是触发本次转换的语义来源（如 `"show"` / `"hide"` / `"ready"` /
    /// `"idle-destroy"`），仅用于日志与前端调试，不参与逻辑分支。
    fn transition(&self, app: &AppHandle, label: &str, phase: LifecyclePhase, reason: &str) {
        let Some(descriptor) = descriptor_for(label) else {
            return;
        };

        let (previous, generation) = self.with_states(|states| {
            let entry = states.entry(label.to_owned()).or_insert(RuntimeState {
                phase: LifecyclePhase::Created,
                generation: 0,
            });

            let previous = entry.phase;

            // 每次进入 Visible 开启新代次，使尚未触发的旧销毁计时器失效。
            if matches!(phase, LifecyclePhase::Visible) {
                entry.generation += 1;
            }

            entry.phase = phase;

            (previous, entry.generation)
        });

        log::debug!(
            "window lifecycle: {label} {previous:?} -> {phase:?} (gen {generation}, reason {reason})"
        );

        // 首次进入 HiddenWarm 的 DestroyWhenIdle 窗口：启动空闲销毁计时器。
        // 重复进入 HiddenWarm（如主窗口隐藏时对已收起的预览窗口再次 hide）不再补计时器：
        // 每代次只需首次进入时启动的那一条，到点按代次校验决定销毁或放弃，
        // 重复启动只会堆积休眠线程。
        if matches!(phase, LifecyclePhase::HiddenWarm)
            && previous != LifecyclePhase::HiddenWarm
            && descriptor.retain_policy == RetainPolicy::DestroyWhenIdle
        {
            schedule_idle_destroy(app, label, generation);
        }

        if !descriptor.emits_lifecycle {
            return;
        }

        let visible = matches!(phase, LifecyclePhase::Visible);
        if let Err(err) = app.emit(
            WINDOW_LIFECYCLE_EVENT,
            LifecyclePayload {
                label,
                phase,
                generation,
                reason,
                visible,
            },
        ) {
            log::error!("emit window lifecycle failed for {label}: {err:?}");
        }
    }
}

impl Default for WindowLifecycleManager {
    fn default() -> Self {
        Self::new()
    }
}

/// 从 `AppHandle` 取 manager；未 `manage` 时返回 `None`（理论上 setup 后恒存在）。
fn manager(app: &AppHandle) -> Option<tauri::State<'_, WindowLifecycleManager>> {
    app.try_state::<WindowLifecycleManager>()
}

/// 窗口显示成功后记录 `Visible`。
pub fn on_shown(app: &AppHandle, label: &str) {
    if let Some(manager) = manager(app) {
        manager.transition(app, label, LifecyclePhase::Visible, "show");
    }
}

/// 窗口隐藏成功后记录 `HiddenWarm`。
pub fn on_hidden(app: &AppHandle, label: &str, reason: &str) {
    if let Some(manager) = manager(app) {
        manager.transition(app, label, LifecyclePhase::HiddenWarm, reason);
    }
}

/// 前端 ready handshake：标记窗口前端已完成基础初始化。
pub fn on_ready(app: &AppHandle, label: &str) {
    if let Some(manager) = manager(app) {
        manager.transition(app, label, LifecyclePhase::Ready, "ready");
    }
}

/// 取窗口的按需重建函数；非 `DestroyWhenIdle` 或未登记窗口返回 `None`。
/// 供 `show_window` 在窗口已销毁时重建 WebView。
pub fn rebuild_fn(label: &str) -> Option<fn(&AppHandle) -> crate::core::Result<()>> {
    descriptor_for(label).and_then(|descriptor| descriptor.build)
}

/// 启动一次性空闲销毁计时器。捕获进入隐藏态时的 `generation`，到点回主线程校验后销毁。
/// 用独立线程 + `thread::sleep`（与 `preview.rs` 同模式），不依赖 async runtime。
fn schedule_idle_destroy(app: &AppHandle, label: &str, generation: u64) {
    let app = app.clone();
    let label = label.to_owned();

    thread::spawn(move || {
        thread::sleep(Duration::from_secs(IDLE_DESTROY_SECS));

        let main_app = app.clone();
        let main_label = label.clone();
        if let Err(err) = app.run_on_main_thread(move || {
            try_destroy_idle(&main_app, &main_label, generation);
        }) {
            log::warn!("idle destroy main-thread dispatch failed for {label}: {err}");
        }
    });
}

/// 计时器到点的销毁判定（主线程）：代次未变且仍处于 HiddenWarm 才销毁，
/// 否则说明窗口已被重新显示或已销毁，放弃本次销毁。
fn try_destroy_idle(app: &AppHandle, label: &str, generation: u64) {
    let Some(manager) = manager(app) else {
        return;
    };

    let proceed = manager.with_states(|states| match states.get(label) {
        Some(state) => state.generation == generation && state.phase == LifecyclePhase::HiddenWarm,
        None => false,
    });

    if !proceed {
        return;
    }

    // 销毁前落盘几何，重建时可恢复上次位置与尺寸。
    if let Err(err) = super::state::save_window_state(app, label) {
        log::warn!("save window state before idle destroy failed for {label}: {err}");
    }

    // nspanel 的 to_panel 用 object_setClass 把 NSWindow 改成动态生成的 panel 子类，再额外
    // retain 一份存进注册表。若带着这个子类让 tao 析构，dealloc 会在 Rust 落地处抛出无法跨
    // FFI 捕获的 ObjC 异常直接 abort（“Rust cannot catch foreign exceptions”）。to_window 是
    // to_panel 的逆操作：还原原始类与 delegate、从注册表移除句柄，使随后的 destroy 与普通窗口
    // （如 preference）走完全相同的释放路径——destroy 与 nspanel 示例用的 close 共用 tao 的
    // close_async 收尾，只少发一个会被 hide-on-close 拦截的 CloseRequested。
    // 非 panel 窗口 get_webview_panel 返回 Err，此处跳过即为 no-op。当前已在主线程，满足 AppKit 要求。
    #[cfg(target_os = "macos")]
    {
        use tauri_nspanel::ManagerExt;

        if let Ok(panel) = app.get_webview_panel(label) {
            let _ = panel.to_window();
        }
    }

    if let Some(window) = app.get_webview_window(label) {
        // 用 `destroy` 而非 `close`：close 会触发 `CloseRequested` 被 hide-on-close 拦截，
        // destroy 直接释放 WebView，绕过拦截。macOS 上 panel 窗口须先经上方 to_window 还原类，
        // 否则带着 swizzle 子类析构会 abort。
        if let Err(err) = window.destroy() {
            log::error!("idle destroy window failed for {label}: {err}");
            return;
        }
    }

    manager.transition(app, label, LifecyclePhase::Destroyed, "idle-destroy");
}
