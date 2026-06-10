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
  ClipboardGroupInput,
  ClipboardGroupRecord,
  ClipboardItemPage,
  ClipboardItemQuery,
  ClipboardKind,
  ClipboardSubKind,
  UpdateNoteResult,
} from "@/types/clipboard";
import type { Settings, SettingsPatch } from "@/types/settings";
import { log } from "@/utils/log";
import { confirmClearClipboardItems } from "./confirmClearClipboardItems";

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

export interface ContextSubmenuAnchor {
  left: number;
  top: number;
  width: number;
  height: number;
}

export interface ContextSubmenuGroupInput {
  checked: boolean;
  id: string;
  label: string;
}

export interface ShowContextSubmenuInput {
  action: ClipboardAction;
  anchor: ContextSubmenuAnchor;
  groups: ContextSubmenuGroupInput[];
  itemId: string;
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
  isSensitive: boolean;
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

export interface CleanCacheResult {
  removedFiles: number;
  removedBytes: number;
  storageUsage: StorageUsage;
}

export interface StorageLocation {
  currentPath: string;
  defaultPath: string;
  isCustom: boolean;
}

export interface ChangeStorageLocationResult {
  location: StorageLocation;
  storageUsage: StorageUsage;
}

export type PreferenceDirectoryTarget = "data" | "logs";
export type BackupExportMode = "encrypted" | "plain";
export type BackupContainerMode = "encrypted" | "plain";
export type BackupReceiveSource = "dragDrop" | "openFile";
export type BackupImportStrategy = "merge" | "overwrite";

export interface ExportHistoryBackupOptions {
  mode: BackupExportMode;
  password?: string;
}

export interface ExportHistoryBackupResult {
  path: string;
  totalBytes: number;
  itemCount: number;
  textCount: number;
  imageCount: number;
  filesCount: number;
  resourceBytes: number;
  exportedAt: string;
  mode: BackupExportMode;
}

export interface InspectHistoryBackupInput {
  path: string;
  source?: BackupReceiveSource;
}

export interface ImportHistoryBackupInput {
  path: string;
  password?: string;
}

export interface ImportHistoryBackupOptions {
  strategy: BackupImportStrategy;
}

export interface ImportHistoryBackupResult {
  strategy: BackupImportStrategy;
  importedItems: number;
  skippedItems: number;
  importedResources: number;
  importedSettings: boolean;
  requiresRestart: boolean;
}

export interface BackupReceivedPayload {
  path: string;
  source: BackupReceiveSource;
  mode: BackupContainerMode;
}

/**
 * 把任意 invoke reject 的值归一化成前端可展示的 `AppError`。
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
 * 暂停全局快捷键注册；录入快捷键期间避免旧绑定被直接触发。
 */
export const suspendGlobalShortcuts = () => {
  return call<void>(
    TAURI_COMMAND.SUSPEND_GLOBAL_SHORTCUTS,
    "commands:labels.suspendGlobalShortcuts",
  );
};

/**
 * 按 Rust 当前设置恢复全局快捷键注册；录入完成、取消或失焦后调用。
 */
export const resumeGlobalShortcuts = () => {
  return call<void>(
    TAURI_COMMAND.RESUME_GLOBAL_SHORTCUTS,
    "commands:labels.resumeGlobalShortcuts",
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
 * 恢复所有偏好默认值；历史记录和资源文件不受影响。
 */
export const resetSettings = async () => {
  const settings = await call<Settings>(
    TAURI_COMMAND.RESET_SETTINGS,
    "commands:labels.resetSettings",
  );

  message.success(i18n.t("commands:messages.settingsReset"));

  return settings;
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
 * 读取当前真实数据目录位置。
 */
export const getStorageLocation = () => {
  return call<StorageLocation>(
    TAURI_COMMAND.GET_STORAGE_LOCATION,
    "commands:labels.loadStorageLocation",
  );
};

/**
 * 将数据迁移到用户选择的父目录下，并热切换运行时数据根。
 */
export const changeStorageLocation = async (targetParentDir: string) => {
  const result = await call<ChangeStorageLocationResult>(
    TAURI_COMMAND.CHANGE_STORAGE_LOCATION,
    "commands:labels.changeStorageLocation",
    { targetParentDir },
  );

  message.success(i18n.t("commands:messages.storageLocationChanged"));

  return result;
};

/**
 * 将数据迁回默认目录，并热切换运行时数据根。
 */
export const resetStorageLocation = async () => {
  const result = await call<ChangeStorageLocationResult>(
    TAURI_COMMAND.RESET_STORAGE_LOCATION,
    "commands:labels.resetStorageLocation",
  );

  message.success(i18n.t("commands:messages.storageLocationReset"));

  return result;
};

/**
 * 清理不再被历史记录或资源索引引用的本地资源缓存。
 */
export const cleanResourceCache = async () => {
  const result = await call<CleanCacheResult>(
    TAURI_COMMAND.CLEAN_RESOURCE_CACHE,
    "commands:labels.cleanCache",
  );

  const messageKey =
    result.removedFiles === 0 && result.removedBytes === 0
      ? "commands:messages.cacheAlreadyClean"
      : "commands:messages.cacheCleaned";

  message.success(
    i18n.t(messageKey, {
      count: result.removedFiles,
      size: formatCommandBytes(result.removedBytes),
    }),
  );

  return result;
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
 * 导出历史数据库、资源和设置为 `.ecopastebak` 备份包。
 */
export const exportHistoryBackup = async (
  targetPath: string,
  options: ExportHistoryBackupOptions,
) => {
  const result = await call<ExportHistoryBackupResult>(
    TAURI_COMMAND.EXPORT_HISTORY_BACKUP,
    "commands:labels.exportBackup",
    {
      options,
      targetPath,
    },
  );

  message.success(
    i18n.t("commands:messages.backupExported", {
      count: result.itemCount,
      size: formatCommandBytes(result.totalBytes),
    }),
  );

  return result;
};

/**
 * 识别 `.ecopastebak` 文件并广播给偏好页导入接收壳。
 */
export const inspectHistoryBackup = (input: InspectHistoryBackupInput) => {
  return call<BackupContainerMode>(
    TAURI_COMMAND.INSPECT_HISTORY_BACKUP,
    "commands:labels.inspectBackup",
    { input },
  );
};

/**
 * 取走偏好窗口重建前 Rust 暂存的备份接收事件，供重建后首屏补发。
 * 偏好窗口空闲销毁后再触发备份打开时，事件无法 push 给尚未挂载的前端，改由此主动拉取。
 * 失败不弹 toast：属内部补发信号，失败只记日志。
 */
export const takePendingBackup = async () => {
  try {
    return await invoke<BackupReceivedPayload | null>(
      TAURI_COMMAND.TAKE_PENDING_BACKUP,
    );
  } catch (error) {
    log.error("take pending backup failed", toAppError(error));

    return null;
  }
};

/**
 * 从 `.ecopastebak` 备份包导入历史和/或设置。
 */
export const importHistoryBackup = async (
  input: ImportHistoryBackupInput,
  options: ImportHistoryBackupOptions,
) => {
  const result = await call<ImportHistoryBackupResult>(
    TAURI_COMMAND.IMPORT_HISTORY_BACKUP,
    "commands:labels.importBackup",
    {
      input,
      options,
    },
  );

  message.success(
    i18n.t(
      result.strategy === "overwrite"
        ? "commands:messages.backupOverwriteImported"
        : "commands:messages.backupImported",
      {
        imported: result.importedItems,
        skipped: result.skippedItems,
      },
    ),
  );

  return result;
};

/**
 * 命令层 toast 使用的轻量字节格式化，避免偏好页工具反向依赖命令入口。
 */
const formatCommandBytes = (bytes: number) => {
  if (bytes < 1024) return `${bytes} B`;

  const units = ["KB", "MB", "GB", "TB"];
  let value = bytes / 1024;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  return `${value.toFixed(value >= 10 ? 1 : 2)} ${units[unitIndex]}`;
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
 * 列出可过滤应用：DB 已知应用加上当前运行中应用。
 */
export const listAllApps = () => {
  return call<ClipboardApp[]>(
    TAURI_COMMAND.LIST_ALL_APPS,
    "commands:labels.loadApps",
  );
};

/**
 * 手动添加一个来源应用，并返回写入后的应用信息。
 */
export const addClipboardAppFromPath = (path: string) => {
  return call<ClipboardApp>(
    TAURI_COMMAND.ADD_CLIPBOARD_APP_FROM_PATH,
    "commands:labels.addApp",
    { path },
  );
};

/**
 * 删除没有被历史记录引用的来源应用，并返回实际删除的应用 id。
 */
export const deleteUnreferencedClipboardApps = (ids: string[]) => {
  return call<string[]>(
    TAURI_COMMAND.DELETE_UNREFERENCED_CLIPBOARD_APPS,
    "commands:labels.deleteApps",
    { ids },
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
 * 列出自定义剪贴板分组；隐藏态由调用方按场景决定是否过滤。
 */
export const listClipboardGroups = () => {
  return call<ClipboardGroupRecord[]>(
    TAURI_COMMAND.LIST_CLIPBOARD_GROUPS,
    "commands:labels.loadClipboardGroups",
  );
};

/**
 * 新建自定义剪贴板分组。
 */
export const createClipboardGroup = async (input: ClipboardGroupInput) => {
  const group = await call<ClipboardGroupRecord>(
    TAURI_COMMAND.CREATE_CLIPBOARD_GROUP,
    "commands:labels.saveClipboardGroup",
    { input },
  );

  message.success(i18n.t("commands:messages.clipboardGroupSaved"));

  return group;
};

/**
 * 更新自定义剪贴板分组。
 */
export const updateClipboardGroup = async (
  id: string,
  input: ClipboardGroupInput,
) => {
  await call<void>(
    TAURI_COMMAND.UPDATE_CLIPBOARD_GROUP,
    "commands:labels.saveClipboardGroup",
    { id, input },
  );

  message.success(i18n.t("commands:messages.clipboardGroupSaved"));
};

/**
 * 保存自定义剪贴板分组的排序和主界面显隐状态。
 */
export const updateClipboardGroupsLayout = async (
  order: string[],
  visibleIds: string[],
) => {
  await call<void>(
    TAURI_COMMAND.UPDATE_CLIPBOARD_GROUPS_LAYOUT,
    "commands:labels.saveClipboardGroupsLayout",
    { input: { order, visibleIds } },
  );

  message.success(i18n.t("commands:messages.clipboardGroupsLayoutSaved"));
};

/**
 * 删除自定义剪贴板分组。
 */
export const deleteClipboardGroup = async (id: string) => {
  await call<void>(
    TAURI_COMMAND.DELETE_CLIPBOARD_GROUP,
    "commands:labels.deleteClipboardGroup",
    { id },
  );

  message.success(i18n.t("commands:messages.clipboardGroupDeleted"));
};

/**
 * 将单条剪贴板记录移动到指定自定义分组。
 */
export const updateClipboardItemGroup = async (id: string, groupId: string) => {
  await call<void>(
    TAURI_COMMAND.UPDATE_CLIPBOARD_ITEM_GROUP,
    "commands:labels.moveToGroup",
    { groupId, id },
  );

  message.success(i18n.t("commands:messages.itemMovedToGroup"));
};

/**
 * 读取 Tauri dialog 选中的 SVG 文件内容。
 */
export const importClipboardGroupSvg = (path: string) => {
  return call<string>(
    TAURI_COMMAND.IMPORT_CLIPBOARD_GROUP_SVG,
    "commands:labels.importClipboardGroupSvg",
    { path },
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
 * `plain` 为显式纯文本动作；默认复制格式由 Rust 按设置与记录类型决定。
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
 * `plain` 为显式纯文本 / 路径粘贴动作；默认粘贴格式由 Rust 按设置与记录类型决定。
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
 * 翻转置顶态；`pinned` 表示本次期望的新状态，用于失败 toast 文案。
 * Rust 返回翻转后的真实状态，调用方据此同步 UI 和列表排序。
 */
export const toggleClipboardItemPinned = async (
  id: string,
  pinned: boolean,
) => {
  const next = await call<boolean>(
    TAURI_COMMAND.TOGGLE_CLIPBOARD_ITEM_PINNED,
    pinned ? "commands:labels.pinItem" : "commands:labels.unpinItem",
    { id },
  );

  message.success(
    i18n.t(
      next ? "commands:messages.itemPinned" : "commands:messages.itemUnpinned",
    ),
  );

  return next;
};

/**
 * 删除条目；命令**不**广播 `clipboard://updated`，调用方需根据返回值本地移除该项。
 * 普通条目、收藏条目与置顶条目分别读取对应保护 / 确认开关。
 * 成功后统一 toast「已删除」。
 */
export const deleteClipboardItem = async (
  id: string,
  isFavorite: boolean,
  isPinned: boolean,
): Promise<boolean> => {
  const contentSettings = settingsState.clipboard?.content;

  if (isFavorite && !(contentSettings?.deleteFavoriteItems ?? false)) {
    return false;
  }

  if (isPinned && !(contentSettings?.deletePinnedItems ?? false)) {
    return false;
  }

  const needConfirm =
    (isFavorite && (contentSettings?.deleteFavoriteConfirm ?? true)) ||
    (isPinned && (contentSettings?.deletePinnedConfirm ?? true)) ||
    (!isFavorite && !isPinned && (contentSettings?.deleteConfirm ?? true));

  if (needConfirm) {
    const ok = await new Promise<boolean>((resolve) => {
      Modal.confirm({
        cancelText: i18n.t("common:actions.cancel"),
        centered: true,
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
 * 清空剪贴板历史；默认保留收藏和置顶，确认选项决定是否连带删除受保护记录。
 */
export const clearClipboardItems = async (): Promise<boolean> => {
  const options = await confirmClearClipboardItems();

  if (!options) return false;

  const removed = await call<number>(
    TAURI_COMMAND.CLEAR_CLIPBOARD_ITEMS,
    "commands:labels.clearClipboardItems",
    {
      deleteFavorites: options.deleteFavorites,
      deletePinned: options.deletePinned,
    },
  );

  message.success(
    i18n.t("commands:messages.clipboardItemsCleared", { count: removed }),
  );

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
 * 按窗口 label 隐藏窗口（偏好窗口、剪贴板窗口等）。
 */
export const hideWindow = (label: string) => {
  return call<void>(TAURI_COMMAND.HIDE_WINDOW, "commands:labels.closeWindow", {
    label,
  });
};

/**
 * 上报当前 WebView 已完成基础初始化，由 Rust 生命周期管理器把窗口推进到 ready 阶段。
 * 失败不弹 toast：ready handshake 属内部信号，失败只记日志，不打扰用户。
 */
export const notifyWindowReady = async (label: string) => {
  try {
    await invoke<void>(TAURI_COMMAND.NOTIFY_WINDOW_READY, { label });
  } catch (error) {
    log.error("notify window ready failed", toAppError(error));
  }
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
 * 临时暂停主窗口自动隐藏，供系统文件选择等原生交互保持主窗口可见。
 */
export const setMainWindowAutoHideSuspended = (suspended: boolean) => {
  return call<void>(
    TAURI_COMMAND.SET_MAIN_WINDOW_AUTO_HIDE_SUSPENDED,
    "commands:labels.setMainWindowAutoHideSuspended",
    {
      suspended,
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
  currentGroupId: string | null,
  isFavorite: boolean,
  isPinned: boolean,
  hasNote: boolean,
) => {
  return call<void>(
    TAURI_COMMAND.POPUP_CLIPBOARD_ITEM_MENU,
    "commands:labels.openMenu",
    {
      input: {
        availableActions,
        currentGroupId,
        hasNote,
        isFavorite,
        isPinned,
        itemId,
      },
    },
  );
};

/**
 * 显示 Windows 自定义右键菜单的二级窗口。
 * 内部菜单生命周期命令失败只记日志，避免 hover 过程中打扰用户。
 */
export const showContextSubmenu = async (input: ShowContextSubmenuInput) => {
  try {
    await invoke<void>(TAURI_COMMAND.SHOW_CONTEXT_SUBMENU, { input });
  } catch (error) {
    log.error("show context submenu failed", toAppError(error));
  }
};

/**
 * 隐藏 Windows 自定义右键菜单的二级窗口。
 */
export const hideContextSubmenu = async () => {
  try {
    await invoke<void>(TAURI_COMMAND.HIDE_CONTEXT_SUBMENU);
  } catch (error) {
    log.error("hide context submenu failed", toAppError(error));
  }
};

/**
 * 隐藏 Windows 自定义右键菜单的一级和二级窗口。
 */
export const hideContextMenus = async () => {
  try {
    await invoke<void>(TAURI_COMMAND.HIDE_CONTEXT_MENUS);
  } catch (error) {
    log.error("hide context menus failed", toAppError(error));
  }
};
