//! 列表项右键菜单（Rust 侧）。
//!
//! 背景：tauri-apps/tauri#9470 描述的 Windows 崩溃/卡顿——前端 `Menu.new` 在 `popup`
//! 后立即被 GC，muda 内部菜单被 drop；用户点击菜单项时事件派发到已释放指针。
//! 解决方式：菜单实例由 Rust 持有（`ClipboardItemMenuState.current`），下次 popup
//! 时再替换，确保事件派发期间菜单始终存活。
//!
//! 业务侧（toast / 二次确认 modal / 列表本地镜像同步）仍在前端 `List.tsx`
//! 维护，本模块只负责「建菜单 + popup + 点击后 emit `clipboard://menu-action` 给前端」。

use std::collections::HashSet;
use std::sync::Mutex;

use anyhow::Context;
use serde::{Deserialize, Serialize};
use tauri::menu::{IsMenuItem, Menu, MenuBuilder, MenuItem, PredefinedMenuItem};
use tauri::{AppHandle, Emitter, Manager, WebviewWindow, Wry};

use crate::core::{AppError, Result};
use crate::window::MAIN_WINDOW_LABEL;

/// 菜单项 id 前缀；`on_menu_event` 按前缀分流到本模块，避免与托盘菜单 id 冲突。
const MENU_PREFIX: &str = "cim::";

/// 前端订阅事件：携带 `{action, itemId}`，由 `List.tsx` 派发到现有处理逻辑。
pub const CLIPBOARD_MENU_ACTION_EVENT: &str = "clipboard://menu-action";

/// 与前端 `ClipboardAction` 对齐（`serde(rename_all = "camelCase")`）。
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum ClipboardMenuAction {
    Paste,
    PasteAsPlainText,
    PasteAsPath,
    Copy,
    OpenLink,
    SendEmail,
    RevealInFinder,
    RevealInExplorer,
    ToggleFavorite,
    EditNote,
    Delete,
}

impl ClipboardMenuAction {
    const ALL: &'static [Self] = &[
        Self::Paste,
        Self::PasteAsPlainText,
        Self::PasteAsPath,
        Self::Copy,
        Self::OpenLink,
        Self::SendEmail,
        Self::RevealInFinder,
        Self::RevealInExplorer,
        Self::ToggleFavorite,
        Self::EditNote,
        Self::Delete,
    ];

    fn id(self) -> &'static str {
        match self {
            Self::Paste => "cim::paste",
            Self::PasteAsPlainText => "cim::pasteAsPlainText",
            Self::PasteAsPath => "cim::pasteAsPath",
            Self::Copy => "cim::copy",
            Self::OpenLink => "cim::openLink",
            Self::SendEmail => "cim::sendEmail",
            Self::RevealInFinder => "cim::revealInFinder",
            Self::RevealInExplorer => "cim::revealInExplorer",
            Self::ToggleFavorite => "cim::toggleFavorite",
            Self::EditNote => "cim::editNote",
            Self::Delete => "cim::delete",
        }
    }

    fn from_id(id: &str) -> Option<Self> {
        Self::ALL.iter().copied().find(|a| a.id() == id)
    }

    fn label(self, is_favorite: bool) -> &'static str {
        match self {
            Self::Paste => "粘贴",
            Self::PasteAsPlainText => "粘贴为纯文本",
            Self::PasteAsPath => "粘贴为路径",
            Self::Copy => "复制",
            Self::OpenLink => "打开链接",
            Self::SendEmail => "发送邮件",
            Self::RevealInFinder => "在访达中显示",
            Self::RevealInExplorer => "在资源管理器中显示",
            Self::ToggleFavorite => {
                if is_favorite {
                    "取消收藏"
                } else {
                    "收藏"
                }
            }
            Self::EditNote => "编辑备注",
            Self::Delete => "删除",
        }
    }

    fn accelerator(self) -> Option<&'static str> {
        match self {
            Self::Paste => Some("Enter"),
            Self::PasteAsPlainText | Self::PasteAsPath => Some("CmdOrCtrl+Enter"),
            Self::ToggleFavorite => Some("CmdOrCtrl+D"),
            Self::Delete => Some("CmdOrCtrl+Backspace"),
            _ => None,
        }
    }
}

/// 视觉分组：组间插入分隔线，组内顺序与组顺序为菜单展示顺序。
const ACTION_GROUPS: &[&[ClipboardMenuAction]] = &[
    &[
        ClipboardMenuAction::Paste,
        ClipboardMenuAction::PasteAsPlainText,
        ClipboardMenuAction::PasteAsPath,
        ClipboardMenuAction::Copy,
    ],
    &[
        ClipboardMenuAction::OpenLink,
        ClipboardMenuAction::SendEmail,
        ClipboardMenuAction::RevealInFinder,
        ClipboardMenuAction::RevealInExplorer,
    ],
    &[
        ClipboardMenuAction::ToggleFavorite,
        ClipboardMenuAction::EditNote,
    ],
    &[ClipboardMenuAction::Delete],
];

/// 当前活跃的菜单 + 正在弹菜单的目标项 id。
/// 持有菜单到下次 popup 时被替换，保证事件派发期间不会 use-after-free。
#[derive(Default)]
pub struct ClipboardItemMenuState {
    current: Mutex<Option<Menu<Wry>>>,
    target_item_id: Mutex<Option<String>>,
}

/// 在 `setup` 中调用：把状态塞进 `AppHandle`，再注册菜单事件分发。
pub fn init(app: &AppHandle) {
    app.manage(ClipboardItemMenuState::default());
}

/// 在主窗口当前光标处弹出列表项右键菜单。
/// 业务参数 `available_actions` 与 `is_favorite` 由前端传入（前端拿到的就是 Rust
/// 端 `compute_available_actions` 的结果，菜单仅做展示过滤与文案翻转）。
///
/// 命令本身**立刻返回**：菜单的构建与弹出都被丢到主线程上异步执行。原因：
/// `WebviewWindow::popup_menu` 在 macOS / Windows 都是模态阻塞调用（菜单关闭前
/// 不返回），同时 `MenuItem::with_id` 等在 macOS 上必须主线程；如果在 tokio
/// worker 上同步等待，会卡住一个 worker 几秒（菜单展示期间），并且前端的
/// `await invoke()` 会一直挂到菜单关闭，期间所有 IPC（事件、其他命令）排队
/// → UI 表现为卡顿。
#[tauri::command]
pub fn popup_clipboard_item_menu(
    app: AppHandle,
    window: WebviewWindow,
    item_id: String,
    available_actions: Vec<ClipboardMenuAction>,
    is_favorite: bool,
) -> Result<()> {
    let state = app
        .try_state::<ClipboardItemMenuState>()
        .ok_or_else(|| AppError::Other(anyhow::anyhow!("ClipboardItemMenuState not managed")))?;

    *state.target_item_id.lock().unwrap() = Some(item_id);

    let app_for_main = app.clone();
    let window_for_main = window.clone();
    window
        .run_on_main_thread(move || {
            let menu = match build_menu(&app_for_main, &available_actions, is_favorite) {
                Ok(m) => m,
                Err(err) => {
                    log::warn!("build clipboard item menu failed: {err}");
                    return;
                }
            };

            let state = app_for_main.state::<ClipboardItemMenuState>();
            // 替换：旧菜单在此 drop。事件已派发完成或菜单已关闭，替换无副作用；
            // 新菜单在锁内被持有，事件派发期间永远存活，规避 muda use-after-free。
            {
                let mut current = state.current.lock().unwrap();
                *current = Some(menu);
            }

            let guard = state.current.lock().unwrap();
            let menu_ref = guard
                .as_ref()
                .expect("menu just stored above and lock is held");
            if let Err(err) = window_for_main.popup_menu(menu_ref) {
                log::warn!("popup clipboard item menu failed: {err}");
            }
        })
        .map_err(|err| {
            AppError::Other(anyhow::anyhow!(
                "schedule popup_clipboard_item_menu on main thread failed: {err}"
            ))
        })
}

fn build_menu(
    app: &AppHandle,
    actions: &[ClipboardMenuAction],
    is_favorite: bool,
) -> Result<Menu<Wry>> {
    let active: HashSet<ClipboardMenuAction> = actions.iter().copied().collect();

    // 先把 owned 项收进 Vec，最后统一传引用给 builder。
    enum Entry {
        Item(MenuItem<Wry>),
        Sep(PredefinedMenuItem<Wry>),
    }
    let mut entries: Vec<Entry> = Vec::new();
    let mut first_group = true;

    for group in ACTION_GROUPS {
        let group_items: Vec<ClipboardMenuAction> = group
            .iter()
            .copied()
            .filter(|a| active.contains(a))
            .collect();
        if group_items.is_empty() {
            continue;
        }

        if !first_group {
            let sep = PredefinedMenuItem::separator(app).context("build separator")?;
            entries.push(Entry::Sep(sep));
        }
        first_group = false;

        for action in group_items {
            let item = MenuItem::with_id(
                app,
                action.id(),
                action.label(is_favorite),
                true,
                action.accelerator(),
            )
            .with_context(|| format!("build menu item {}", action.id()))?;
            entries.push(Entry::Item(item));
        }
    }

    let refs: Vec<&dyn IsMenuItem<Wry>> = entries
        .iter()
        .map(|e| match e {
            Entry::Item(i) => i as &dyn IsMenuItem<Wry>,
            Entry::Sep(s) => s as &dyn IsMenuItem<Wry>,
        })
        .collect();

    let mut builder = MenuBuilder::new(app);
    for r in refs {
        builder = builder.item(r);
    }
    builder
        .build()
        .context("build clipboard item menu")
        .map_err(Into::into)
}

/// 全局菜单事件分发：托盘菜单已由 tray 自己的 `on_menu_event` 接管，本回调只处理
/// 列表项右键菜单（id 以 `cim::` 开头）。
pub fn handle_menu_event(app: &AppHandle, menu_id: &str) {
    if !menu_id.starts_with(MENU_PREFIX) {
        return;
    }
    let Some(action) = ClipboardMenuAction::from_id(menu_id) else {
        log::warn!("unknown clipboard menu id: {menu_id}");
        return;
    };

    let Some(state) = app.try_state::<ClipboardItemMenuState>() else {
        log::warn!("ClipboardItemMenuState missing on menu event");
        return;
    };

    let item_id = state.target_item_id.lock().unwrap().clone();
    let Some(item_id) = item_id else {
        log::warn!("no target item id recorded for menu action {menu_id}");
        return;
    };

    // 派发回前端：业务逻辑（toast/确认弹窗/本地镜像同步）由 `List.tsx` 收口。
    // 只发给主窗口，避免偏好窗等无关 webview 也收到。
    let payload = MenuActionPayload { action, item_id };
    let Some(main) = app.get_webview_window(MAIN_WINDOW_LABEL) else {
        log::warn!("main window missing on clipboard menu dispatch");
        return;
    };
    if let Err(err) = main.emit(CLIPBOARD_MENU_ACTION_EVENT, payload) {
        log::warn!("emit {CLIPBOARD_MENU_ACTION_EVENT} failed: {err}");
    }
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct MenuActionPayload {
    action: ClipboardMenuAction,
    item_id: String,
}
