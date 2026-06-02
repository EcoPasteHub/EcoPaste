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
import { settingsState } from "@/stores/settings";
import type {
  ClipboardAction,
  ClipboardItemPage,
  ClipboardItemQuery,
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
  label: string,
  args?: Record<string, unknown>,
): Promise<T> => {
  try {
    return await invoke<T>(command, args);
  } catch (error) {
    const appError = toAppError(error);

    log.error(`invoke ${command} failed`, appError);
    message.error(`${label}失败：${appError.message}`);

    throw appError;
  }
};

/**
 * 拉取设置首屏快照；后续刷新走 `settings://updated` 事件。
 */
export const getSettings = () => {
  return call<Settings>(TAURI_COMMAND.GET_SETTINGS, "加载设置");
};

/**
 * 提交设置补丁；Rust 落盘后广播 `settings://updated` 由各窗口回灌镜像。
 */
export const updateSettings = (patch: SettingsPatch) => {
  return call<Settings>(TAURI_COMMAND.UPDATE_SETTINGS, "保存设置", { patch });
};

/**
 * 列表查询；返回顶页项 + 总数 + `hasMore`，供列表分页与 Footer 共用。
 */
export const listClipboardItems = (query: ClipboardItemQuery) => {
  return call<ClipboardItemPage>(
    TAURI_COMMAND.LIST_CLIPBOARD_ITEMS,
    "加载列表",
    { query },
  );
};

/**
 * 打开条目 URL：`mailto = true` 时 Rust 侧自动裹 `mailto:`。
 * 用于右键菜单「打开链接 / 发送邮件」。
 */
export const openClipboardItemLink = (id: string, mailto: boolean) => {
  return call<void>(TAURI_COMMAND.OPEN_CLIPBOARD_ITEM_LINK, "打开链接", {
    id,
    mailto,
  });
};

/**
 * 在系统文件管理器中定位条目对应文件；Rust 侧自动按 kind 提路径（files 取首个，text 取 content）。
 */
export const revealClipboardItem = (id: string) => {
  return call<void>(TAURI_COMMAND.REVEAL_CLIPBOARD_ITEM, "打开位置", { id });
};

/**
 * 写回剪贴板（不模拟粘贴）：右键菜单「复制」走此命令。
 * 成功后统一 toast「已复制」，调用方无需再处理。
 */
export const writeToClipboard = async (id: string, plain: boolean) => {
  await call<void>(TAURI_COMMAND.WRITE_TO_CLIPBOARD, "复制", { id, plain });

  message.success("已复制");
};

/**
 * 「写回剪贴板 + 隐藏主窗口 + 模拟系统粘贴」的组合命令。
 * 回车 / 数字快捷键 / 右键菜单全部走这里。
 */
export const pasteClipboardItem = (id: string, plain: boolean) => {
  return call<void>(TAURI_COMMAND.PASTE_CLIPBOARD_ITEM, "粘贴", { id, plain });
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
    favorite ? "收藏" : "取消收藏",
    { id },
  );

  message.success(next ? "已收藏" : "已取消收藏");

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
        cancelText: "取消",
        content: "删除后无法恢复，确定删除这条记录吗？",
        okButtonProps: { danger: true },
        okText: "删除",
        onCancel: () => resolve(false),
        onOk: () => resolve(true),
        title: "删除记录",
      });
    });

    if (!ok) return false;
  }

  await call<void>(TAURI_COMMAND.DELETE_CLIPBOARD_ITEM, "删除", { id });

  message.success("已删除");

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
    "保存备注",
    { id, note },
  );

  message.success(result.autoFavorited ? "已保存并收藏" : "已保存");

  return result;
};

/**
 * 按窗口 label 显示窗口（偏好窗口、剪贴板窗口等）。
 */
export const showWindow = (label: string) => {
  return call<void>(TAURI_COMMAND.SHOW_WINDOW, "打开窗口", { label });
};

/**
 * 设置主窗口固定态：Rust 侧立即生效（影响 resign_key / 外部点击自动隐藏逻辑）。
 */
export const setMainWindowPinned = (pinned: boolean) => {
  return call<void>(TAURI_COMMAND.SET_MAIN_WINDOW_PINNED, "固定窗口", {
    pinned,
  });
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
  return call<void>(TAURI_COMMAND.POPUP_CLIPBOARD_ITEM_MENU, "打开菜单", {
    availableActions,
    isFavorite,
    itemId,
  });
};
