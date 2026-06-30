//! 统一窗口生命周期管理器。
//!
//! 持有所有窗口的 phase / generation 状态，把 show / hide / toggle / close / 销毁重建
//! 路径收口到这里。在 `window://visibility` 之外广播单一 `window://lifecycle` 事件
//! （带 `phase` 字段），前端据此镜像每个窗口的生命周期阶段。
//!
//! 销毁策略：`DestroyWhenIdle` 窗口（当前为 preference 与 clipboard-preview）隐藏后
//! 启动空闲计时器，超过用户设置的空闲秒数仍隐藏则销毁 WebView 释放资源；再次打开时
//! 经 descriptor 的 `build` 重建（preference 走 `show_window`，preview 走预览模块按需 ensure
//! 建窗）。`generation` 用于让「显示又隐藏」期间的过期计时器自动失效。

mod descriptor;

pub use descriptor::{descriptor_for, descriptors, RetainPolicy};

use std::collections::{HashMap, HashSet};
use std::sync::Mutex;
use std::thread;
use std::time::{Duration, Instant};

use tauri::{AppHandle, Emitter, Manager};

use crate::settings::SettingsStore;

/// 非剪贴板窗口隐藏空闲超过此时长后销毁 WebView。
pub const DEFAULT_IDLE_DESTROY_SECS: u64 = 60;
/// 非剪贴板窗口最短销毁空闲时间，避免用户把窗口设置成近乎瞬时销毁导致重建闪烁。
const MIN_IDLE_DESTROY_SECS: u64 = 5;
/// 非剪贴板窗口最长销毁空闲时间，限制异常配置导致计时器长时间悬挂。
const MAX_IDLE_DESTROY_SECS: u64 = 24 * 60 * 60;
/// 剪贴板窗口隐藏后进入 dormant 的宽限时间；剪贴板窗口不销毁，只暂停非必要工作。
const CLIPBOARD_DORMANT_SECS: u64 = 5;
/// 销毁前给前端保存草稿 / 申请 keepalive 的时间。
const BEFORE_DESTROY_DEADLINE_MS: u64 = 500;
/// keepalive lease 的默认兜底时长。
const DEFAULT_KEEPALIVE_TIMEOUT_MS: u64 = 30_000;
/// keepalive lease 的最长兜底时长，防止异常路径永久阻止销毁。
const MAX_KEEPALIVE_TIMEOUT_MS: u64 = 10 * 60 * 1000;

/// 生命周期阶段。
#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub enum LifecyclePhase {
    /// 窗口仅存在于 descriptor 声明中，尚未创建 WebView 实例。
    NotCreated,
    /// 窗口已登记但尚未收到任何转换：作为首次转换日志的「来源」哨兵阶段。
    Created,
    /// 前端完成基础初始化，可安全 show / emit payload。
    Ready,
    /// 窗口可见。
    Visible,
    /// 刚隐藏，保留实例，允许快速恢复；`DestroyWhenIdle` 窗口在此阶段计时销毁。
    HiddenWarm,
    /// 剪贴板窗口隐藏一小段时间后进入休眠：保留实例，但前端应暂停非必要刷新。
    Dormant,
    /// 非剪贴板窗口空闲到期，已通知前端 before-destroy，等待最后保护状态确认。
    DestroyPending,
    /// WebView 已销毁，仅保留 descriptor；再次打开时重建。
    Destroyed,
}

/// 单个窗口的运行时生命周期状态。
struct RuntimeState {
    phase: LifecyclePhase,
    /// 单调递增代次：每次从非可见状态进入 `Visible` 自增。空闲销毁计时器捕获进入
    /// `HiddenWarm` 时的代次，到点若代次已变，说明计时器已过期，直接放弃销毁。
    generation: u64,
    hidden_at: Option<Instant>,
    last_active_at: Instant,
    dirty_owners: HashSet<String>,
    keepalive_leases: HashMap<String, KeepaliveLease>,
}

/// 前端申请的窗口保活租约。`expires_at` 是异常路径兜底，不是业务完成信号。
struct KeepaliveLease {
    reason: String,
    expires_at: Instant,
}

enum DestroyCheck {
    Proceed,
    Protected,
    Stale,
}

impl RuntimeState {
    /// 构造窗口运行时状态；窗口首次被 manager 观察到时创建。
    fn new() -> Self {
        let now = Instant::now();

        Self {
            dirty_owners: HashSet::new(),
            generation: 0,
            hidden_at: None,
            keepalive_leases: HashMap::new(),
            last_active_at: now,
            phase: LifecyclePhase::Created,
        }
    }

    /// 应用一次生命周期阶段转换，并维护计时器判定所需的派生状态。
    fn transition_phase(&mut self, phase: LifecyclePhase, now: Instant) {
        let previous = self.phase;

        if matches!(phase, LifecyclePhase::Visible) && previous != LifecyclePhase::Visible {
            self.generation += 1;
            self.hidden_at = None;
            self.last_active_at = now;
        }

        if matches!(phase, LifecyclePhase::HiddenWarm) && previous != LifecyclePhase::HiddenWarm {
            self.hidden_at = Some(now);
            self.last_active_at = now;
        }

        if matches!(phase, LifecyclePhase::Destroyed) {
            self.hidden_at = None;
            self.dirty_owners.clear();
            self.keepalive_leases.clear();
        }

        self.phase = phase;
    }

    /// 清理过期 keepalive，返回仍然有效的数量。
    fn purge_expired_leases(&mut self) -> usize {
        let now = Instant::now();
        self.keepalive_leases.retain(|owner, lease| {
            let active = lease.expires_at > now;
            if !active {
                log::warn!(
                    "window keepalive expired: owner={owner}, reason={}",
                    lease.reason
                );
            }

            active
        });

        self.keepalive_leases.len()
    }

    /// 当前窗口是否有前端声明的未保存状态或保活租约。
    fn destroy_blocked(&mut self) -> bool {
        self.purge_expired_leases();

        !self.dirty_owners.is_empty() || !self.keepalive_leases.is_empty()
    }
}

/// 生命周期事件名。与 `window://visibility` 并存：visibility 只表达布尔可见性，
/// lifecycle 表达完整阶段。
const WINDOW_LIFECYCLE_EVENT: &str = "window://lifecycle";
const WINDOW_BEFORE_DESTROY_EVENT: &str = "window://before-destroy";

#[derive(Clone, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct LifecyclePayload<'a> {
    label: &'a str,
    phase: LifecyclePhase,
    generation: u64,
    reason: &'a str,
    visible: bool,
}

#[derive(Clone, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct BeforeDestroyPayload<'a> {
    label: &'a str,
    generation: u64,
    deadline_ms: u64,
}

/// 单个窗口的生命周期调试快照。
#[derive(Clone, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct LifecycleSnapshot {
    pub label: String,
    pub phase: LifecyclePhase,
    pub generation: u64,
    pub visible: bool,
    pub retain_policy: &'static str,
    pub dirty_owner_count: usize,
    pub keepalive_count: usize,
    pub hidden_for_ms: Option<u128>,
    pub last_active_ago_ms: u128,
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
            let entry = states
                .entry(label.to_owned())
                .or_insert_with(RuntimeState::new);

            let previous = entry.phase;
            let now = Instant::now();

            entry.transition_phase(phase, now);

            (previous, entry.generation)
        });

        log::debug!(
            "window lifecycle: {label} {previous:?} -> {phase:?} (gen {generation}, reason {reason})"
        );

        // 首次进入 HiddenWarm 的 DestroyWhenIdle 窗口：启动空闲销毁计时器。
        // 重复进入 HiddenWarm（如剪贴板窗口隐藏时对已收起的预览窗口再次 hide）不再补计时器：
        // 每代次只需首次进入时启动的那一条，到点按代次校验决定销毁或放弃，
        // 重复启动只会堆积休眠线程。
        if matches!(phase, LifecyclePhase::HiddenWarm)
            && previous != LifecyclePhase::HiddenWarm
            && descriptor.retain_policy == RetainPolicy::DestroyWhenIdle
            && lightweight_mode_enabled(app)
        {
            schedule_idle_destroy(app, label, generation, idle_destroy_secs(app));
        }

        if matches!(phase, LifecyclePhase::HiddenWarm)
            && previous != LifecyclePhase::HiddenWarm
            && label == super::CLIPBOARD_WINDOW_LABEL
            && lightweight_mode_enabled(app)
        {
            schedule_clipboard_dormant(app, label, generation);
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
    if descriptor_for(label).is_none() {
        return;
    }

    if let Some(manager) = manager(app) {
        let should_transition = manager.with_states(|states| {
            let entry = states
                .entry(label.to_owned())
                .or_insert_with(RuntimeState::new);

            !matches!(
                entry.phase,
                LifecyclePhase::Visible
                    | LifecyclePhase::HiddenWarm
                    | LifecyclePhase::Dormant
                    | LifecyclePhase::DestroyPending
            )
        });

        if should_transition {
            manager.transition(app, label, LifecyclePhase::Ready, "ready");
        } else {
            log::debug!("window ready acknowledged without phase change: {label}");
        }
    }
}

/// 设置窗口 dirty owner；任一 owner 未清除时，空闲销毁会被延后。
pub fn set_dirty(app: &AppHandle, label: &str, owner: &str, dirty: bool) {
    if descriptor_for(label).is_none() {
        return;
    }

    if let Some(manager) = manager(app) {
        manager.with_states(|states| {
            let entry = states
                .entry(label.to_owned())
                .or_insert_with(RuntimeState::new);

            if dirty {
                entry.dirty_owners.insert(owner.to_owned());
            } else {
                entry.dirty_owners.remove(owner);
            }

            log::debug!(
                "window dirty: {label} owner={owner} dirty={dirty} count={}",
                entry.dirty_owners.len()
            );
        });
    }
}

/// 申请窗口保活租约；租约到期会自动失效，前端正常完成后应主动 release。
pub fn acquire_keepalive(
    app: &AppHandle,
    label: &str,
    owner: &str,
    reason: &str,
    timeout_ms: Option<u64>,
) {
    if descriptor_for(label).is_none() {
        return;
    }

    let timeout = timeout_ms
        .unwrap_or(DEFAULT_KEEPALIVE_TIMEOUT_MS)
        .clamp(1_000, MAX_KEEPALIVE_TIMEOUT_MS);
    let lease = KeepaliveLease {
        expires_at: Instant::now() + Duration::from_millis(timeout),
        reason: reason.to_owned(),
    };

    if let Some(manager) = manager(app) {
        manager.with_states(|states| {
            let entry = states
                .entry(label.to_owned())
                .or_insert_with(RuntimeState::new);
            entry.keepalive_leases.insert(owner.to_owned(), lease);

            log::debug!(
                "window keepalive acquired: {label} owner={owner} reason={reason} timeoutMs={timeout}"
            );
        });
    }
}

/// 释放窗口保活租约；不存在时 no-op。
pub fn release_keepalive(app: &AppHandle, label: &str, owner: &str) {
    if descriptor_for(label).is_none() {
        return;
    }

    if let Some(manager) = manager(app) {
        manager.with_states(|states| {
            let Some(entry) = states.get_mut(label) else {
                return;
            };
            entry.keepalive_leases.remove(owner);

            log::debug!(
                "window keepalive released: {label} owner={owner} count={}",
                entry.keepalive_leases.len()
            );
        });
    }
}

/// 返回所有登记窗口的生命周期调试快照。
pub fn snapshot(app: &AppHandle) -> Vec<LifecycleSnapshot> {
    let now = Instant::now();
    let Some(manager) = manager(app) else {
        return Vec::new();
    };

    manager.with_states(|states| {
        descriptors()
            .iter()
            .map(|descriptor| {
                let window = app.get_webview_window(descriptor.label);
                let visible = window
                    .as_ref()
                    .and_then(|window| window.is_visible().ok())
                    .unwrap_or(false);
                let Some(state) = states.get_mut(descriptor.label) else {
                    return LifecycleSnapshot {
                        dirty_owner_count: 0,
                        generation: 0,
                        hidden_for_ms: None,
                        keepalive_count: 0,
                        label: descriptor.label.to_owned(),
                        last_active_ago_ms: 0,
                        phase: if window.is_some() {
                            LifecyclePhase::Created
                        } else {
                            LifecyclePhase::NotCreated
                        },
                        retain_policy: descriptor.retain_policy.as_str(),
                        visible,
                    };
                };
                let keepalive_count = state.purge_expired_leases();

                LifecycleSnapshot {
                    dirty_owner_count: state.dirty_owners.len(),
                    generation: state.generation,
                    hidden_for_ms: state
                        .hidden_at
                        .map(|instant| now.saturating_duration_since(instant).as_millis()),
                    keepalive_count,
                    label: descriptor.label.to_owned(),
                    last_active_ago_ms: now
                        .saturating_duration_since(state.last_active_at)
                        .as_millis(),
                    phase: snapshot_phase(state, window.is_some()),
                    retain_policy: descriptor.retain_policy.as_str(),
                    visible,
                }
            })
            .collect()
    })
}

/// 解析诊断快照展示阶段；`Created` 仅在真实 WebView 存在时表示已创建。
fn snapshot_phase(state: &RuntimeState, has_window: bool) -> LifecyclePhase {
    if !has_window && state.phase == LifecyclePhase::Created && state.generation == 0 {
        return LifecyclePhase::NotCreated;
    }

    state.phase
}

/// 取窗口的按需重建函数；非 `DestroyWhenIdle` 或未登记窗口返回 `None`。
/// 供 `show_window` 在窗口已销毁时重建 WebView。
pub fn rebuild_fn(label: &str) -> Option<fn(&AppHandle) -> crate::core::Result<()>> {
    descriptor_for(label).and_then(|descriptor| descriptor.build)
}

/// 启动一次性剪贴板窗口 dormant 计时器。捕获进入隐藏态时的 `generation`，到点回主线程校验。
fn schedule_clipboard_dormant(app: &AppHandle, label: &str, generation: u64) {
    let app = app.clone();
    let label = label.to_owned();

    thread::spawn(move || {
        thread::sleep(Duration::from_secs(CLIPBOARD_DORMANT_SECS));

        let main_app = app.clone();
        let main_label = label.clone();
        if let Err(err) = app.run_on_main_thread(move || {
            try_enter_dormant(&main_app, &main_label, generation);
        }) {
            log::warn!("main dormant main-thread dispatch failed for {label}: {err}");
        }
    });
}

/// 计时器到点的 dormant 判定：剪贴板窗口仍隐藏且代次未变才进入 dormant。
fn try_enter_dormant(app: &AppHandle, label: &str, generation: u64) {
    if !lightweight_mode_enabled(app) {
        return;
    }

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

    manager.transition(app, label, LifecyclePhase::Dormant, "idle-dormant");
}

/// 启动一次性空闲销毁计时器。捕获进入隐藏态时的 `generation`，到点回主线程校验后销毁。
/// 用独立线程 + `thread::sleep`（与 `preview.rs` 同模式），不依赖 async runtime。
fn schedule_idle_destroy(app: &AppHandle, label: &str, generation: u64, timeout_secs: u64) {
    let app = app.clone();
    let label = label.to_owned();

    thread::spawn(move || {
        thread::sleep(Duration::from_secs(timeout_secs));

        let main_app = app.clone();
        let main_label = label.clone();
        if let Err(err) = app.run_on_main_thread(move || {
            try_destroy_idle(&main_app, &main_label, generation);
        }) {
            log::warn!("idle destroy main-thread dispatch failed for {label}: {err}");
        }
    });
}

/// 计时器到点的销毁判定（主线程）：代次未变且仍处于 HiddenWarm 才进入 DestroyPending，
/// 否则说明窗口已被重新显示或已销毁，放弃本次销毁。
fn try_destroy_idle(app: &AppHandle, label: &str, generation: u64) {
    if !lightweight_mode_enabled(app) {
        return;
    }

    let Some(manager) = manager(app) else {
        return;
    };

    let check = manager.with_states(|states| match states.get_mut(label) {
        Some(state) => {
            if state.generation != generation || state.phase != LifecyclePhase::HiddenWarm {
                return DestroyCheck::Stale;
            }

            if state.destroy_blocked() {
                DestroyCheck::Protected
            } else {
                DestroyCheck::Proceed
            }
        }
        None => DestroyCheck::Stale,
    });

    match check {
        DestroyCheck::Proceed => {}
        DestroyCheck::Protected => {
            schedule_idle_destroy(app, label, generation, idle_destroy_secs(app));
            return;
        }
        DestroyCheck::Stale => return,
    }

    manager.transition(app, label, LifecyclePhase::DestroyPending, "before-destroy");
    emit_before_destroy(app, label, generation);
    schedule_destroy_after_deadline(app, label, generation);
}

/// 广播销毁前事件，给前端保存草稿或申请 keepalive 的短暂窗口。
fn emit_before_destroy(app: &AppHandle, label: &str, generation: u64) {
    if let Err(err) = app.emit(
        WINDOW_BEFORE_DESTROY_EVENT,
        BeforeDestroyPayload {
            deadline_ms: BEFORE_DESTROY_DEADLINE_MS,
            generation,
            label,
        },
    ) {
        log::error!("emit window before destroy failed for {label}: {err:?}");
    }
}

/// 销毁前宽限到期后回主线程完成最终校验与释放。
fn schedule_destroy_after_deadline(app: &AppHandle, label: &str, generation: u64) {
    let app = app.clone();
    let label = label.to_owned();

    thread::spawn(move || {
        thread::sleep(Duration::from_millis(BEFORE_DESTROY_DEADLINE_MS));

        let main_app = app.clone();
        let main_label = label.clone();
        if let Err(err) = app.run_on_main_thread(move || {
            finish_destroy_idle(&main_app, &main_label, generation);
        }) {
            log::warn!("finish destroy main-thread dispatch failed for {label}: {err}");
        }
    });
}

/// 最终销毁判定：仍处于 DestroyPending、代次未变且没有 dirty / keepalive 才释放 WebView。
fn finish_destroy_idle(app: &AppHandle, label: &str, generation: u64) {
    if !lightweight_mode_enabled(app) {
        return;
    }

    let Some(manager) = manager(app) else {
        return;
    };

    let check = manager.with_states(|states| match states.get_mut(label) {
        Some(state) => {
            if state.generation != generation || state.phase != LifecyclePhase::DestroyPending {
                return DestroyCheck::Stale;
            }

            if state.destroy_blocked() {
                DestroyCheck::Protected
            } else {
                DestroyCheck::Proceed
            }
        }
        None => DestroyCheck::Stale,
    });

    match check {
        DestroyCheck::Proceed => {}
        DestroyCheck::Protected => {
            manager.transition(app, label, LifecyclePhase::HiddenWarm, "destroy-protected");
            return;
        }
        DestroyCheck::Stale => return,
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

/// 当前是否启用轻量模式。设置缺失时默认启用，匹配 `Settings::default()`。
fn lightweight_mode_enabled(app: &AppHandle) -> bool {
    app.try_state::<SettingsStore>()
        .map(|settings| settings.snapshot().clipboard.window.lightweight_mode)
        .unwrap_or(true)
}

/// 从设置读取非剪贴板窗口空闲销毁秒数，并做边界收敛。
fn idle_destroy_secs(app: &AppHandle) -> u64 {
    app.try_state::<SettingsStore>()
        .map(|settings| {
            u64::from(settings.snapshot().clipboard.window.idle_destroy_seconds)
                .clamp(MIN_IDLE_DESTROY_SECS, MAX_IDLE_DESTROY_SECS)
        })
        .unwrap_or(DEFAULT_IDLE_DESTROY_SECS)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dirty_owner_blocks_destroy_until_cleared() {
        let mut state = RuntimeState::new();

        state.dirty_owners.insert("draft".to_owned());

        assert!(state.destroy_blocked());

        state.dirty_owners.clear();

        assert!(!state.destroy_blocked());
    }

    #[test]
    fn keepalive_blocks_destroy_until_released() {
        let mut state = RuntimeState::new();

        state.keepalive_leases.insert(
            "dialog".to_owned(),
            KeepaliveLease {
                expires_at: Instant::now() + Duration::from_secs(10),
                reason: "file-dialog".to_owned(),
            },
        );

        assert!(state.destroy_blocked());

        state.keepalive_leases.remove("dialog");

        assert!(!state.destroy_blocked());
    }

    #[test]
    fn expired_keepalive_no_longer_blocks_destroy() {
        let mut state = RuntimeState::new();

        state.keepalive_leases.insert(
            "dialog".to_owned(),
            KeepaliveLease {
                expires_at: Instant::now() - Duration::from_secs(1),
                reason: "file-dialog".to_owned(),
            },
        );

        assert!(!state.destroy_blocked());
        assert!(state.keepalive_leases.is_empty());
    }

    #[test]
    fn repeated_visible_transition_keeps_generation() {
        let mut state = RuntimeState::new();
        let now = Instant::now();

        state.transition_phase(LifecyclePhase::Visible, now);
        state.transition_phase(LifecyclePhase::Visible, now + Duration::from_secs(1));

        assert_eq!(state.generation, 1);

        state.transition_phase(LifecyclePhase::HiddenWarm, now + Duration::from_secs(2));
        state.transition_phase(LifecyclePhase::Visible, now + Duration::from_secs(3));

        assert_eq!(state.generation, 2);
    }

    #[test]
    fn created_state_without_webview_snapshots_as_not_created() {
        let state = RuntimeState::new();

        assert_eq!(snapshot_phase(&state, false), LifecyclePhase::NotCreated);
        assert_eq!(snapshot_phase(&state, true), LifecyclePhase::Created);
    }

    #[test]
    fn destroyed_state_without_webview_stays_destroyed() {
        let mut state = RuntimeState::new();

        state.phase = LifecyclePhase::Destroyed;

        assert_eq!(snapshot_phase(&state, false), LifecyclePhase::Destroyed);
    }
}
