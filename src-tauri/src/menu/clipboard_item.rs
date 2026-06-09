//! 列表项右键菜单（Rust 侧）。
//!
//! - **macOS**：用原生 muda `Menu` + `popup_menu`。NSPanel 不 resignKey，菜单
//!   弹出不会偷走前台焦点；菜单实例由 [`native::ClipboardItemMenuState`] 持有
//!   到下次 popup 前替换，规避 tauri-apps/tauri#9470 的 muda use-after-free。
//! - **Windows**：muda 的 `TrackPopupMenu` 必须把菜单 owner 拉到前台，会把用户
//!   原本聚焦的目标 App（如资源管理器重命名编辑框）挤掉焦点。改用自定义
//!   webview 窗（`focusable: false`，不偷焦点）实现，逻辑在
//!   [`super::context_window`]。
//!
//! 业务侧（toast / 二次确认 modal / 列表本地镜像同步）仍在前端 `List.tsx`
//! 维护，本模块只负责「弹菜单 + 点击后 emit `clipboard://menu-action` 给前端」。

use serde::{Deserialize, Serialize};
use tauri::AppHandle;

use crate::core::Result;
use crate::settings::Language;

/// 前端订阅事件：携带 `{action, itemId}`，由 `List.tsx` 派发到现有处理逻辑。
#[cfg(target_os = "macos")]
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
    TogglePinned,
    EditNote,
    Delete,
}

impl ClipboardMenuAction {
    /// 返回当前语言下的菜单文案；切换类动作按当前状态翻转。
    pub(super) fn label(
        self,
        lang: Language,
        is_favorite: bool,
        is_pinned: bool,
        has_note: bool,
    ) -> &'static str {
        use crate::i18n::clipboard_menu::Key;

        let key = match self {
            Self::Paste => Key::Paste,
            Self::PasteAsPlainText => Key::PasteAsPlainText,
            Self::PasteAsPath => Key::PasteAsPath,
            Self::Copy => Key::Copy,
            Self::OpenLink => Key::OpenLink,
            Self::SendEmail => Key::SendEmail,
            Self::RevealInFinder => Key::RevealInFinder,
            Self::RevealInExplorer => Key::RevealInExplorer,
            Self::ToggleFavorite => {
                if is_favorite {
                    Key::Unfavorite
                } else {
                    Key::Favorite
                }
            }
            Self::TogglePinned => {
                if is_pinned {
                    Key::UnpinItem
                } else {
                    Key::PinItem
                }
            }
            Self::EditNote => {
                if has_note {
                    Key::EditNote
                } else {
                    Key::AddNote
                }
            }
            Self::Delete => Key::Delete,
        };

        crate::i18n::clipboard_menu::label(lang, key)
    }

    /// 加速键文案（muda 与前端菜单共用 `"CmdOrCtrl+X"` 平台无关写法）。
    pub(super) fn accelerator(self) -> Option<&'static str> {
        match self {
            Self::Paste => Some("Enter"),
            Self::PasteAsPlainText | Self::PasteAsPath => Some("CmdOrCtrl+Enter"),
            Self::ToggleFavorite => Some("CmdOrCtrl+D"),
            Self::TogglePinned => Some("CmdOrCtrl+T"),
            Self::EditNote => Some("CmdOrCtrl+M"),
            Self::Delete => Some("CmdOrCtrl+Backspace"),
            _ => None,
        }
    }
}

/// 视觉分组：组间插入分隔线，组内顺序与组顺序即菜单展示顺序。两端共用。
pub(super) const ACTION_GROUPS: &[&[ClipboardMenuAction]] = &[
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
        ClipboardMenuAction::TogglePinned,
        ClipboardMenuAction::EditNote,
    ],
    &[ClipboardMenuAction::Delete],
];

/// 菜单点击后 emit 给前端的 payload。Windows 自定义菜单窗也复用这个结构发回
/// 主窗，前端 `List.tsx` 只需订阅一次。
#[cfg(target_os = "macos")]
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub(super) struct MenuActionPayload {
    pub action: ClipboardMenuAction,
    pub item_id: String,
}

// ============================================================================
// macOS：muda 原生菜单
// ============================================================================

#[cfg(target_os = "macos")]
mod native {
    use std::collections::HashSet;
    use std::sync::Mutex;

    use anyhow::Context;
    use tauri::menu::{IsMenuItem, Menu, MenuBuilder, MenuItem, PredefinedMenuItem};
    use tauri::{AppHandle, Emitter, Manager, Wry};

    use crate::core::{AppError, Result};
    use crate::settings::Language;
    use crate::window::MAIN_WINDOW_LABEL;

    use super::{
        ClipboardMenuAction, MenuActionPayload, ACTION_GROUPS, CLIPBOARD_MENU_ACTION_EVENT,
    };

    /// 菜单项 id 前缀；`on_menu_event` 按前缀分流到本模块，避免与托盘菜单 id 冲突。
    const MENU_PREFIX: &str = "cim::";

    impl ClipboardMenuAction {
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
                Self::TogglePinned => "cim::togglePinned",
                Self::EditNote => "cim::editNote",
                Self::Delete => "cim::delete",
            }
        }

        fn from_id(id: &str) -> Option<Self> {
            const ALL: &[ClipboardMenuAction] = &[
                ClipboardMenuAction::Paste,
                ClipboardMenuAction::PasteAsPlainText,
                ClipboardMenuAction::PasteAsPath,
                ClipboardMenuAction::Copy,
                ClipboardMenuAction::OpenLink,
                ClipboardMenuAction::SendEmail,
                ClipboardMenuAction::RevealInFinder,
                ClipboardMenuAction::RevealInExplorer,
                ClipboardMenuAction::ToggleFavorite,
                ClipboardMenuAction::TogglePinned,
                ClipboardMenuAction::EditNote,
                ClipboardMenuAction::Delete,
            ];
            ALL.iter().copied().find(|a| a.id() == id)
        }
    }

    /// 当前活跃的菜单 + 正在弹菜单的目标项 id；菜单持有到下次 popup 前替换，
    /// 保证事件派发期间不会 use-after-free。
    #[derive(Default)]
    pub(super) struct ClipboardItemMenuState {
        current: Mutex<Option<Menu<Wry>>>,
        target_item_id: Mutex<Option<String>>,
    }

    pub(super) fn init(app: &AppHandle) {
        app.manage(ClipboardItemMenuState::default());
    }

    /// 命令本身**立刻返回**：菜单的构建与弹出都被丢到主线程上异步执行。
    /// muda 的 `MenuItem::with_id` 与 `popup_menu` 都必须主线程，且
    /// `popup_menu` 在菜单关闭前不返回（模态阻塞）。
    pub(super) fn popup(
        app: &AppHandle,
        item_id: String,
        available_actions: Vec<ClipboardMenuAction>,
        is_favorite: bool,
        is_pinned: bool,
        has_note: bool,
    ) -> Result<()> {
        let state = app.try_state::<ClipboardItemMenuState>().ok_or_else(|| {
            AppError::Other(anyhow::anyhow!("ClipboardItemMenuState not managed"))
        })?;
        *state.target_item_id.lock().unwrap() = Some(item_id);

        let window = app
            .get_webview_window(MAIN_WINDOW_LABEL)
            .ok_or_else(|| AppError::Other(anyhow::anyhow!("main window missing")))?;

        let app_for_main = app.clone();
        let window_for_main = window.clone();
        let lang = crate::i18n::current_language(app);
        window
            .run_on_main_thread(move || {
                let menu = match build_menu(
                    &app_for_main,
                    &available_actions,
                    lang,
                    is_favorite,
                    is_pinned,
                    has_note,
                ) {
                    Ok(m) => m,
                    Err(err) => {
                        log::warn!("build clipboard item menu failed: {err}");
                        return;
                    }
                };

                let state = app_for_main.state::<ClipboardItemMenuState>();
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
        lang: Language,
        is_favorite: bool,
        is_pinned: bool,
        has_note: bool,
    ) -> Result<Menu<Wry>> {
        let active: HashSet<ClipboardMenuAction> = actions.iter().copied().collect();

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
                    action.label(lang, is_favorite, is_pinned, has_note),
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

    pub(super) fn handle_event(app: &AppHandle, menu_id: &str) {
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

        let payload = MenuActionPayload { action, item_id };
        let Some(main) = app.get_webview_window(MAIN_WINDOW_LABEL) else {
            log::warn!("main window missing on clipboard menu dispatch");
            return;
        };
        if let Err(err) = main.emit(CLIPBOARD_MENU_ACTION_EVENT, payload) {
            log::warn!("emit {CLIPBOARD_MENU_ACTION_EVENT} failed: {err}");
        }
    }
}

// ============================================================================
// 跨平台入口
// ============================================================================

/// setup 阶段调用：macOS 注册 muda 菜单状态；Windows 由 [`super::context_window::init`]
/// 单独建窗，这里 no-op。
#[allow(unused_variables)]
pub fn init(app: &AppHandle) {
    #[cfg(target_os = "macos")]
    native::init(app);
}

/// 在当前光标处弹出列表项右键菜单。`available_actions` / `is_favorite` 由
/// 前端传入（来自 `ClipboardItem.availableActions` 与当前状态字段）。
#[tauri::command]
pub async fn popup_clipboard_item_menu(
    app: AppHandle,
    item_id: String,
    available_actions: Vec<ClipboardMenuAction>,
    is_favorite: bool,
    is_pinned: bool,
    has_note: bool,
) -> Result<()> {
    #[cfg(target_os = "macos")]
    {
        native::popup(
            &app,
            item_id,
            available_actions,
            is_favorite,
            is_pinned,
            has_note,
        )
    }

    #[cfg(target_os = "windows")]
    {
        super::context_window::show_for_clipboard_item(
            &app,
            item_id,
            &available_actions,
            is_favorite,
            is_pinned,
            has_note,
        )
    }
}

/// 全局菜单事件分发入口（注册在 `on_menu_event` 上）。Windows 不经此路径。
#[allow(unused_variables)]
pub fn handle_menu_event(app: &AppHandle, menu_id: &str) {
    #[cfg(target_os = "macos")]
    native::handle_event(app, menu_id);
}
