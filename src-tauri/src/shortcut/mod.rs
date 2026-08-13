//! 全局快捷键：注册由 Rust 主导，前端在偏好设置里改完通过 `update_settings` 触发重注册。
//!
//! 配置来自 `settings::Shortcuts`。本模块只负责 OS 级注册——`paste_plain`
//! 是窗口内交互（前端 `useKeyPress`），不在这里处理。

use std::sync::{LazyLock, Mutex};
use std::time::Duration;

use serde::Serialize;
use tauri::{AppHandle, Emitter, Manager};
use tauri_plugin_global_shortcut::{GlobalShortcutExt, Shortcut, ShortcutEvent, ShortcutState};

use crate::core::{AppError, Result};
use crate::settings::{SettingsStore, Shortcuts};
use crate::window::{self, CLIPBOARD_WINDOW_LABEL, PREFERENCE_WINDOW_LABEL};

#[cfg(target_os = "windows")]
mod win_v;

/// 与前端 `src/constants/events.ts` 的 `TAURI_EVENT.SHORTCUT_CONFLICT` 一一对应。
/// 载荷是本轮 [`apply`] 的完整冲突集合（`Vec<ShortcutConflict>`），空集合不发。
pub const CONFLICT_EVENT: &str = "shortcut://conflict";
const RESUME_DEBOUNCE: Duration = Duration::from_millis(160);

/// 最近一次 [`apply`] 的注册冲突集合，供偏好窗口挂载后主动拉取。
///
/// 冲突绝大多数发生在启动时注册（快捷键已被别的应用占用），此时偏好窗口根本不存在；
/// 偏好窗口又是空闲可销毁的，直接 `emit` 会丢给尚未挂载的前端（与 backup 接收、
/// 定位高亮同源）。故每轮 apply 都把结果写入此 slot，由前端经
/// `take_pending_shortcut_conflicts` 主动拉取。
///
/// 语义是「覆盖」而非「追加」：每轮 apply 产出的都是当前完整冲突集，
/// 写入空集合即表示冲突已解决，能顺带清掉过期的待提示。
static PENDING_CONFLICTS: LazyLock<Mutex<Vec<ShortcutConflict>>> =
    LazyLock::new(|| Mutex::new(Vec::new()));

#[derive(Debug, Clone, Serialize)]
pub struct ShortcutConflict {
    pub action: &'static str,
    pub binding: String,
    pub reason: String,
}

#[derive(Default)]
pub struct ShortcutManager {
    active: Mutex<Vec<(&'static str, Shortcut)>>,
    pause: Mutex<ShortcutPause>,
}

#[derive(Default)]
struct ShortcutPause {
    resume_epoch: u64,
    suspend_count: usize,
}

impl ShortcutPause {
    /// 记录一次暂停请求，并返回是否需要实际注销已注册快捷键。
    fn suspend(&mut self) -> bool {
        self.resume_epoch += 1;
        self.suspend_count += 1;

        self.suspend_count == 1
    }

    /// 释放一次暂停请求；返回 `None` 表示没有对应的暂停可释放。
    fn resume(&mut self) -> Option<bool> {
        if self.suspend_count == 0 {
            return None;
        }

        self.suspend_count -= 1;

        Some(self.suspend_count == 0)
    }

    /// 开启新的恢复世代，用于让更早的延迟恢复任务失效。
    fn next_resume_epoch(&mut self) -> u64 {
        self.resume_epoch += 1;
        self.resume_epoch
    }

    /// 判断延迟恢复任务是否仍对当前暂停状态有效。
    fn allows_resume(&self, epoch: u64) -> bool {
        self.resume_epoch == epoch && self.suspend_count == 0
    }

    /// 判断全局快捷键是否处于暂停态。
    fn suspended(&self) -> bool {
        self.suspend_count > 0
    }
}

pub fn init(app: &AppHandle, shortcuts: &Shortcuts) -> Result<()> {
    app.manage(ShortcutManager::default());
    apply(app, shortcuts)
}

/// 暂停所有已注册的全局快捷键；用于前端录入快捷键期间避免旧绑定被触发。
pub fn suspend(app: &AppHandle) -> Result<()> {
    let manager = app.state::<ShortcutManager>();
    let should_unregister = {
        let mut pause = manager.pause.lock().expect("shortcut state poisoned");
        pause.suspend()
    };

    if should_unregister {
        unregister_active(app)?;
    }

    Ok(())
}

/// 释放一次全局快捷键暂停；只有所有录入器都结束后才按最新设置恢复注册。
pub fn resume(app: &AppHandle) -> Result<()> {
    let manager = app.state::<ShortcutManager>();
    let should_schedule = {
        let mut pause = manager.pause.lock().expect("shortcut state poisoned");

        match pause.resume() {
            Some(should_schedule) => should_schedule,
            None => {
                log::warn!("resume global shortcuts called without active suspend");

                return Ok(());
            }
        }
    };

    if should_schedule {
        schedule_resume(app);
    }

    Ok(())
}

/// 全量替换：先取消上一轮注册，再逐个注册新项。注册失败仅 emit 不中断其它项。
pub fn apply(app: &AppHandle, shortcuts: &Shortcuts) -> Result<()> {
    unregister_active(app)?;

    if is_suspended(app) {
        return Ok(());
    }

    let desired: [(&'static str, &str); 2] = [
        ("open_clipboard", &shortcuts.open_clipboard),
        ("open_preference", &shortcuts.open_preference),
    ];

    #[cfg(target_os = "windows")]
    win_v::set_enabled(app, shortcuts.win_v);

    let mut active = Vec::new();
    let mut conflicts = Vec::new();
    for (action, binding) in desired {
        if binding.trim().is_empty() {
            continue;
        }
        match register_one(app, action, binding) {
            Ok(shortcut) => active.push((action, shortcut)),
            Err(err) => {
                log::warn!("register shortcut {action}={binding} failed: {err}");
                conflicts.push(ShortcutConflict {
                    action,
                    binding: binding.into(),
                    reason: err.to_string(),
                });
            }
        }
    }

    *app.state::<ShortcutManager>()
        .active
        .lock()
        .expect("shortcut state poisoned") = active;

    dispatch_conflicts(app, conflicts);

    Ok(())
}

/// 投递本轮注册冲突：始终写入 pending slot（空集合即清除过期提示），
/// 偏好窗口存活时再额外推一次事件，让用户改完快捷键当场就能看到反馈。
///
/// 两条路径都走是有意为之：事件覆盖「窗口开着」的即时反馈，pending 覆盖
/// 「启动时冲突」与「窗口空闲销毁后重开」。冲突未解决时重开偏好页再提示一次是正确的
/// ——那个快捷键此刻确实仍然没注册上。
fn dispatch_conflicts(app: &AppHandle, conflicts: Vec<ShortcutConflict>) {
    let preference_alive = app.get_webview_window(PREFERENCE_WINDOW_LABEL).is_some();

    if !conflicts.is_empty() && preference_alive {
        if let Err(err) = app.emit_to(PREFERENCE_WINDOW_LABEL, CONFLICT_EVENT, &conflicts) {
            log::warn!("emit shortcut conflict event failed: {err}");
        }
    }

    set_pending_conflicts(conflicts);
}

/// 覆盖式写入待提示的冲突集合（只保留最近一轮 apply 的结果）。
fn set_pending_conflicts(conflicts: Vec<ShortcutConflict>) {
    let mut guard = PENDING_CONFLICTS.lock().unwrap_or_else(|poisoned| {
        log::error!("pending shortcut conflicts mutex poisoned on set, recovering");
        poisoned.into_inner()
    });
    *guard = conflicts;
}

/// 取走并清空待提示的冲突集合，供偏好窗口挂载后首屏补发。
pub fn take_pending_conflicts() -> Vec<ShortcutConflict> {
    let mut guard = PENDING_CONFLICTS.lock().unwrap_or_else(|poisoned| {
        log::error!("pending shortcut conflicts mutex poisoned on take, recovering");
        poisoned.into_inner()
    });
    std::mem::take(&mut *guard)
}

/// 判断是否仍有录入器持有全局快捷键暂停。
fn is_suspended(app: &AppHandle) -> bool {
    app.state::<ShortcutManager>()
        .pause
        .lock()
        .expect("shortcut state poisoned")
        .suspended()
}

/// 延迟恢复全局快捷键；若用户立即切到另一个录入器，新 suspend 会让本次恢复失效。
fn schedule_resume(app: &AppHandle) {
    let app = app.clone();
    let epoch = {
        let manager = app.state::<ShortcutManager>();
        let mut pause = manager.pause.lock().expect("shortcut state poisoned");
        pause.next_resume_epoch()
    };

    tauri::async_runtime::spawn(async move {
        tokio::time::sleep(RESUME_DEBOUNCE).await;

        if !should_run_scheduled_resume(&app, epoch) {
            return;
        }

        let settings = app.state::<SettingsStore>().snapshot();
        if let Err(err) = apply(&app, &settings.shortcuts) {
            log::warn!("resume delayed global shortcuts failed: {err}");
        }
    });
}

/// 只有仍处于同一恢复世代且暂停计数为 0 时，延迟恢复任务才允许执行。
fn should_run_scheduled_resume(app: &AppHandle, epoch: u64) -> bool {
    let manager = app.state::<ShortcutManager>();
    let pause = manager.pause.lock().expect("shortcut state poisoned");

    pause.allows_resume(epoch)
}

/// 取消当前轮所有已注册快捷键，并清空内部 active 状态。
fn unregister_active(app: &AppHandle) -> Result<()> {
    let plugin = app.global_shortcut();
    let manager = app.state::<ShortcutManager>();

    let previous = {
        let mut guard = manager.active.lock().expect("shortcut state poisoned");
        std::mem::take(&mut *guard)
    };
    for (_, shortcut) in &previous {
        if let Err(err) = plugin.unregister(*shortcut) {
            log::warn!("unregister previous shortcut failed: {err:?}");
        }
    }

    Ok(())
}

fn register_one(app: &AppHandle, action: &'static str, binding: &str) -> Result<Shortcut> {
    let plugin = app.global_shortcut();
    let shortcut: Shortcut = binding
        .parse()
        .map_err(|err| AppError::Other(anyhow::anyhow!("parse shortcut {binding}: {err}")))?;

    if plugin.is_registered(shortcut) {
        plugin
            .unregister(shortcut)
            .map_err(|err| AppError::Other(anyhow::anyhow!(err)))?;
    }

    plugin
        .on_shortcut(shortcut, move |app, _scut, event| {
            handle_event(app, action, event);
        })
        .map_err(|err| AppError::Other(anyhow::anyhow!(err)))?;
    Ok(shortcut)
}

fn handle_event(app: &AppHandle, action: &'static str, event: ShortcutEvent) {
    // Pressed 触发一次即可（Released 是按键松开），避免 toggle 在按下/松开各执行一次回弹。
    if !matches!(event.state(), ShortcutState::Pressed) {
        return;
    }
    let label = match action {
        "open_clipboard" => CLIPBOARD_WINDOW_LABEL,
        "open_preference" => PREFERENCE_WINDOW_LABEL,
        _ => return,
    };
    if let Err(err) = window::toggle_window(app, label) {
        log::warn!("toggle window via shortcut {action} failed: {err}");
    }
}
