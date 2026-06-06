/**
 * 前端唯一的 Tauri 命令调用入口（对应 Rust `src-tauri/src/commands/` 各模块）。
 *
 * 约定：
 * - 每个 `#[tauri::command]` 在此文件**只**有一个对应的 TS 包装函数，命名与 Rust 函数同名转 camelCase。
 * - 调用方一律 `import { foo } from "@/commands"`，**禁止**裸调 `invoke` 或引用 `TAURI_COMMAND` 常量。
 * - 错误处理在本文件统一收口：失败时 log + antd `message.error("xxx 失败：${error}")`，
 *   然后再 rethrow。调用方按需用 `try/catch` 决定成功后做什么，**不要再写错误 toast**。
 *
 * 注：用 antd v6 静态 `message` API，不读 ConfigProvider 主题——错误 toast 一致即可，
 * 不必跟随主题精修。希望集成主题再换 `App.useApp()` 拿到的实例 + 模块级 holder。
 */

import { invoke } from "@tauri-apps/api/core";
import { Modal, message } from "antd";
import { TAURI_COMMAND } from "@/constants/commands";
import i18n from "@/i18n";
import { settingsState } from "@/stores/settings";
import type {
  ClipboardAction,
  ClipboardApp,
  ClipboardItemPage,
  ClipboardItemQuery,
  ClipboardKind,
  ClipboardSubKind,
  UpdateNoteResult,
} from "@/types/clipboard";
import type { Settings, SettingsPatch } from "@/types/settings";
import { log } from "@/utils/log";

/**
 * Rust 端 `AppError` 序列化后的形状：`kind` 用于按变体分流，`message` 给用户看。
 */
interface AppError {
  kind: string;
  message: string;
}

export interface PreviewAnchorRect {
  left: number;
  pointerY?: number;
  top: number;
  width: number;
  height: number;
}

export interface ClipboardPreviewState {
  requestId: number;
  sessionId: number;
  itemId: string;
  anchor: PreviewAnchorRect;
  scaleFactor: number;
  workArea: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
  mainWindow: {
    x: number;
    y: number;
    width: number;
    height: number;
  } | null;
  layout: ClipboardPreviewLayout;
}

export interface ClipboardPreviewRect {
  left: number;
  top: number;
  width: number;
  height: number;
}

export type ClipboardPreviewPlacement = "right" | "left" | "bottom" | "top";

export interface ClipboardPreviewLayout {
  overlayRect: ClipboardPreviewRect;
  sourceRect: ClipboardPreviewRect;
  panelRect: ClipboardPreviewRect;
  placement: ClipboardPreviewPlacement;
}

export interface ClipboardPreviewFileEntry {
  path: string;
  name: string;
  isDir: boolean;
  isImage: boolean;
  exists: boolean;
  size: number | null;
  iconPath?: string;
}

export interface ClipboardPreviewPayload {
  id: string;
  kind: ClipboardKind;
  subKind: ClipboardSubKind | null;
  updatedAt: string;
  text: string | null;
  imagePath: string | null;
  imageWidth: number | null;
  imageHeight: number | null;
  size: number | null;
  imageExists: boolean;
  files: ClipboardPreviewFileEntry[];
  totalFiles: number;
}

export interface StorageUsage {
  totalBytes: number;
  databaseBytes: number;
  resourcesBytes: number;
  settingsBytes: number;
}

export type PreferenceDirectoryTarget = "data" | "logs";

/**
 * 把任意 invoke reject 的值归一化成 `AppError`：兼容旧路径（字符串）和未来扩展。
 */
const toAppError = (error: unknown): AppError => {
  if (
    typeof error === "object" &&
    error !== null &&
    "kind" in error &&
    "message" in error
  ) {
    return error as AppError;
  }

  return { kind: "Unknown", message: String(error) };
};

/**
 * invoke 的通用包装：失败 → log + toast + rethrow。
 * `label` 用于 toast 文案（"xxx 失败：message"）。
 */
const call = async <T>(
  command: string,
  labelKey: string,
  args?: Record<string, unknown>,
): Promise<T> => {
  try {
    return await invoke<T>(command, args);
  } catch (error) {
    const appError = toAppError(error);

    log.error(`invoke ${command} failed`, appError);
    message.error(
      i18n.t("commands:error", {
        label: i18n.t(labelKey),
        message: appError.message,
      }),
    );

    throw appError;
  }
};

/**
 * 拉取设置首屏快照；后续刷新走 `settings://updated` 事件。
 */
export const getSettings = () => {
  return call<Settings>(
    TAURI_COMMAND.GET_SETTINGS,
    "commands:labels.loadSettings",
  );
};

/**
 * 提交设置补丁；Rust 落盘后广播 `settings://updated` 由各窗口回灌镜像。
 */
export const updateSettings = (patch: SettingsPatch) => {
  return call<Settings>(
    TAURI_COMMAND.UPDATE_SETTINGS,
    "commands:labels.saveSettings",
    { patch },
  );
};

/**
 * 统计本地数据库、资源缓存与设置文件的占用。
 */
export const getStorageUsage = () => {
  return call<StorageUsage>(
    TAURI_COMMAND.GET_STORAGE_USAGE,
    "commands:labels.loadStorageUsage",
  );
};

/**
 * 打开偏好页固定本地目录：数据目录或日志目录。
 */
export const openPreferenceDirectory = (target: PreferenceDirectoryTarget) => {
  return call<void>(
    TAURI_COMMAND.OPEN_PREFERENCE_DIRECTORY,
    "commands:labels.openDirectory",
    {
      target,
    },
  );
};

/**
 * 使用系统默认浏览器打开经过 Rust 侧校验的外部网页。
 */
export const openExternalUrl = (url: string) => {
  return call<void>(
    TAURI_COMMAND.OPEN_EXTERNAL_URL,
    "commands:labels.openLink",
    { url },
  );
};

/**
 * 查询系统自启动真实状态（auto-launch 后端）。
 */
export const getAutostart = () => {
  return call<boolean>(
    TAURI_COMMAND.GET_AUTOSTART,
    "commands:labels.loadAutostart",
  );
};

/**
 * 设置系统自启动真实状态；偏好页需与 `general.autoStart` 一起更新。
 */
export const setAutostart = (enabled: boolean) => {
  return call<void>(
    TAURI_COMMAND.SET_AUTOSTART,
    "commands:labels.setAutostart",
    {
      enabled,
    },
  );
};

/**
 * 查询本次是否由自启动参数唤起。
 */
export const isLaunchedViaAutostart = () => {
  return call<boolean>(
    TAURI_COMMAND.IS_LAUNCHED_VIA_AUTOSTART,
    "commands:labels.loadLaunchSource",
  );
};

/**
 * 列出系统扫描与历史监听中已知的全部来源应用。
 */
export const listAllApps = () => {
  return call<ClipboardApp[]>(
    TAURI_COMMAND.LIST_ALL_APPS,
    "commands:labels.loadApps",
  );
};

/**
 * 重新扫描应用发现目录，并返回更新后的应用列表。
 */
export const refreshApps = () => {
  return call<ClipboardApp[]>(
    TAURI_COMMAND.REFRESH_APPS,
    "commands:labels.refreshApps",
  );
};

/**
 * 列表查询；返回顶页项 + 总数 + `hasMore`，供列表分页与 Footer 共用。
 */
export const listClipboardItems = (query: ClipboardItemQuery) => {
  return call<ClipboardItemPage>(
    TAURI_COMMAND.LIST_CLIPBOARD_ITEMS,
    "commands:labels.loadClipboardList",
    { query },
  );
};

/**
 * 打开条目 URL：`mailto = true` 时 Rust 侧自动裹 `mailto:`。
 * 用于右键菜单「打开链接 / 发送邮件」。
 */
export const openClipboardItemLink = (id: string, mailto: boolean) => {
  return call<void>(
    TAURI_COMMAND.OPEN_CLIPBOARD_ITEM_LINK,
    "commands:labels.openLink",
    {
      id,
      mailto,
    },
  );
};

/**
 * 在系统文件管理器中定位条目对应文件；Rust 侧自动按 kind 提路径（files 取首个，text 取 content）。
 */
export const revealClipboardItem = (id: string) => {
  return call<void>(
    TAURI_COMMAND.REVEAL_CLIPBOARD_ITEM,
    "commands:labels.reveal",
    { id },
  );
};

/**
 * 写回剪贴板（不模拟粘贴）：右键菜单「复制」走此命令。
 * 成功后统一 toast「已复制」，调用方无需再处理。
 */
export const writeToClipboard = async (id: string, plain: boolean) => {
  await call<void>(TAURI_COMMAND.WRITE_TO_CLIPBOARD, "commands:labels.copy", {
    id,
    plain,
  });

  message.success(i18n.t("commands:messages.copied"));
};

/**
 * 「写回剪贴板 + 隐藏主窗口 + 模拟系统粘贴」的组合命令。
 * 回车 / 数字快捷键 / 右键菜单全部走这里。
 */
export const pasteClipboardItem = (id: string, plain: boolean) => {
  return call<void>(
    TAURI_COMMAND.PASTE_CLIPBOARD_ITEM,
    "commands:labels.paste",
    { id, plain },
  );
};

/**
 * 启动一次 OS 级 drag-out：把条目拖出主窗口到外部应用。
 *
 * - Files / Image：拖出为文件，预览用 OS 原生图标。
 * - Text（含 HTML / RTF 富格式）：接收方按偏好选格式；Rust 端用文本首几行
 *   现场渲染的 PNG 作预览，缺失则退回来源 app 图标。
 *
 * macOS 立即返回（drop 由 OS 异步处理）；Windows 会 await 至 drop 完成。
 * 失败已在 `call` 内统一 toast，调用方一般不需要再处理。
 */
export const startDragClipboardItem = (id: string) => {
  return call<void>(
    TAURI_COMMAND.START_DRAG_CLIPBOARD_ITEM,
    "commands:labels.drag",
    { id },
  );
};

/**
 * 翻转收藏态；`favorite` 表示本次期望的新状态（用于 toast 文案）。
 * Rust 返回翻转后的真实状态，调用方据此同步 UI。
 * 成功后统一 toast「已收藏 / 已取消收藏」，失败也按意图分开「收藏失败 / 取消收藏失败」。
 */
export const toggleClipboardItemFavorite = async (
  id: string,
  favorite: boolean,
) => {
  const next = await call<boolean>(
    TAURI_COMMAND.TOGGLE_CLIPBOARD_ITEM_FAVORITE,
    favorite
      ? "commands:labels.toggleFavorite"
      : "commands:labels.cancelFavorite",
    { id },
  );

  message.success(
    i18n.t(
      next
        ? "commands:messages.favoriteAdded"
        : "commands:messages.favoriteRemoved",
    ),
  );

  return next;
};

/**
 * 删除条目；命令**不**广播 `clipboard://updated`，调用方需根据返回值本地移除该项。
 * 设置 `clipboard.content.deleteConfirm = true` 时弹二次确认 modal；用户取消则 resolve false 且不调 Rust。
 * 成功后统一 toast「已删除」。
 */
export const deleteClipboardItem = async (id: string): Promise<boolean> => {
  if (settingsState.clipboard?.content?.deleteConfirm) {
    const ok = await new Promise<boolean>((resolve) => {
      Modal.confirm({
        cancelText: i18n.t("common:actions.cancel"),
        content: i18n.t("commands:deleteConfirm.content"),
        okButtonProps: { danger: true },
        okText: i18n.t("common:actions.delete"),
        onCancel: () => resolve(false),
        onOk: () => resolve(true),
        title: i18n.t("commands:deleteConfirm.title"),
      });
    });

    if (!ok) return false;
  }

  await call<void>(
    TAURI_COMMAND.DELETE_CLIPBOARD_ITEM,
    "commands:labels.delete",
    { id },
  );

  message.success(i18n.t("commands:messages.deleted"));

  return true;
};

/**
 * 更新备注；Rust 统一 trim + 空串归一为 `null`，返回归一化后的 `note` 与 `autoFavorited`。
 * 调用方用返回的 `note` 回填本地镜像，避免「输入纯空白时镜像非空但 DB 为 NULL」的漂移。
 * 成功后统一 toast：触发 auto-favorite 时「已保存并收藏」，否则「已保存」。
 */
export const updateClipboardItemNote = async (
  id: string,
  note: string | null,
) => {
  const result = await call<UpdateNoteResult>(
    TAURI_COMMAND.UPDATE_CLIPBOARD_ITEM_NOTE,
    "commands:labels.saveNote",
    { id, note },
  );

  message.success(
    i18n.t(
      result.autoFavorited
        ? "commands:messages.noteSavedAndFavorited"
        : "commands:messages.noteSaved",
    ),
  );

  return result;
};

/**
 * 按窗口 label 显示窗口（偏好窗口、剪贴板窗口等）。
 */
export const showWindow = (label: string) => {
  return call<void>(TAURI_COMMAND.SHOW_WINDOW, "commands:labels.openWindow", {
    label,
  });
};

/**
 * 显示或隐藏 macOS Dock / Windows 任务栏图标。
 */
export const showTaskbarIcon = (visible: boolean) => {
  return call<void>(
    TAURI_COMMAND.SHOW_TASKBAR_ICON,
    "commands:labels.setTaskbarIcon",
    {
      visible,
    },
  );
};

/**
 * 设置主窗口固定态：Rust 侧立即生效（影响 resign_key / 外部点击自动隐藏逻辑）。
 */
export const setMainWindowPinned = (pinned: boolean) => {
  return call<void>(
    TAURI_COMMAND.SET_MAIN_WINDOW_PINNED,
    "commands:labels.setMainWindowPinned",
    {
      pinned,
    },
  );
};

/**
 * 打开或重定向剪贴板系统级预览 overlay。
 * `anchor` 是主窗口 webview client 坐标中的列表项矩形。
 */
export const showClipboardPreview = (
  itemId: string,
  anchor: PreviewAnchorRect,
) => {
  return call<ClipboardPreviewState | null>(
    TAURI_COMMAND.SHOW_CLIPBOARD_PREVIEW,
    "commands:labels.openPreview",
    { anchor, itemId },
  );
};

/**
 * 关闭剪贴板系统级预览 overlay。
 */
export const closeClipboardPreview = () => {
  return call<void>(
    TAURI_COMMAND.CLOSE_CLIPBOARD_PREVIEW,
    "commands:labels.closePreview",
  );
};

/**
 * 预览窗口首屏补拉最近一次状态。
 */
export const getClipboardPreviewState = () => {
  return call<ClipboardPreviewState | null>(
    TAURI_COMMAND.GET_CLIPBOARD_PREVIEW_STATE,
    "commands:labels.loadPreviewState",
  );
};

/**
 * 读取预览窗口 Content Viewer 所需的归一化 payload。
 */
export const getClipboardPreviewPayload = (itemId: string) => {
  return call<ClipboardPreviewPayload | null>(
    TAURI_COMMAND.GET_CLIPBOARD_PREVIEW_PAYLOAD,
    "commands:labels.loadPreviewContent",
    { itemId },
  );
};

/**
 * 播放一次复制成功提示音，供偏好设置页试听。
 */
export const playCopySound = () => {
  return call<void>(
    TAURI_COMMAND.PLAY_COPY_SOUND,
    "commands:labels.playCopySound",
  );
};

/**
 * 在主窗口当前光标处弹出列表项右键菜单（菜单实例由 Rust 持有）。
 *
 * 点击菜单项后 Rust 会 emit `clipboard://menu-action` 携带 `{action, itemId}`，
 * 由 `List.tsx` 单点订阅后派发到既有处理逻辑（toast / 确认 modal / 本地镜像同步）。
 *
 * 把菜单生命周期搬到 Rust 是为了规避 tauri-apps/tauri#9470：前端 `Menu.new` 在
 * `popup` 后立即被 GC 会导致 Windows muda 点击崩溃/卡顿。
 */
export const popupClipboardItemMenu = (
  itemId: string,
  availableActions: ClipboardAction[],
  isFavorite: boolean,
) => {
  return call<void>(
    TAURI_COMMAND.POPUP_CLIPBOARD_ITEM_MENU,
    "commands:labels.openMenu",
    {
      availableActions,
      isFavorite,
      itemId,
    },
  );
};
