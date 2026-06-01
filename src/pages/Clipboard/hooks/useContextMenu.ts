import { invoke } from "@tauri-apps/api/core";
import {
  Menu,
  type MenuItemOptions,
  type PredefinedMenuItemOptions,
} from "@tauri-apps/api/menu";
import { openUrl, revealItemInDir } from "@tauri-apps/plugin-opener";
import { App } from "antd";
import type { MouseEvent } from "react";
import { TAURI_COMMAND } from "@/constants/commands";
import { settingsState } from "@/stores/settings";
import type { ClipboardItem } from "@/types/clipboard";
import { isMac } from "@/utils/is";
import { log } from "@/utils/log";

interface UseContextMenuProps {
  item: ClipboardItem;
  /**
   * 删除成功后通知列表移除该项（删除命令不广播 clipboard://updated，靠本地更新）。
   */
  onRemoved: (id: string) => void;
  /**
   * 收藏切换成功后通知列表同步 isFavorite；favorite 分组下据此把该项移除。
   */
  onFavoriteToggled: (id: string, isFavorite: boolean) => void;
  /**
   * 打开备注编辑弹窗，弹窗由列表层单例持有。
   */
  onEditNote: (item: ClipboardItem) => void;
}

const SEPARATOR: PredefinedMenuItemOptions = { item: "Separator" };

/**
 * 列表项右键菜单：返回 `handleContextMenu`，绑到卡片根节点的 `onContextMenu`，
 * 右键时用 Tauri 原生菜单（`Menu.popup`）按 kind / subKind 弹出。
 * 粘贴 / 复制走后端命令；打开链接 / 发邮件 / 文件管理器显示需要原始完整值
 * （列表视图 text 类 content 被置空、summary 又截断），故先用
 * get_clipboard_item_content 按 id 拉完整值。删除是否二次确认由
 * clipboard.content.deleteConfirm 设置决定。
 */
export const useContextMenu = (props: UseContextMenuProps) => {
  const { item, onRemoved, onFavoriteToggled, onEditNote } = props;
  const { kind, subKind, isFavorite } = item;

  const { message, modal } = App.useApp();

  /**
   * 取条目完整 content：text 类在列表视图被裁剪，按 id 回查；files 类未裁剪，直接用。
   */
  const resolveContent = async () => {
    if (kind === "files") return item.content;

    const content = await invoke<string | null>(
      TAURI_COMMAND.GET_CLIPBOARD_ITEM_CONTENT,
      { id: item.id },
    );

    return content ?? "";
  };

  const paste = async (plain: boolean) => {
    try {
      await invoke(TAURI_COMMAND.PASTE_CLIPBOARD_ITEM, { id: item.id, plain });
    } catch (error) {
      log.error("paste clipboard item failed", error);
      message.error("粘贴失败");
    }
  };

  const copy = async () => {
    try {
      await invoke(TAURI_COMMAND.WRITE_TO_CLIPBOARD, {
        id: item.id,
        plain: false,
      });

      message.success("已复制");
    } catch (error) {
      log.error("write to clipboard failed", error);
      message.error("复制失败");
    }
  };

  const openLink = async (mailto: boolean) => {
    try {
      const value = (await resolveContent()).trim();

      if (!value) return;

      await openUrl(mailto ? `mailto:${value}` : value);
    } catch (error) {
      log.error("open url failed", error);
      message.error("打开失败");
    }
  };

  const reveal = async () => {
    try {
      const value = await resolveContent();
      // files 的 content 是换行分隔的多路径，取首个定位。
      const target = kind === "files" ? value.split("\n")[0] : value.trim();

      if (!target) return;

      await revealItemInDir(target);
    } catch (error) {
      log.error("reveal item in dir failed", error);
      message.error("打开位置失败");
    }
  };

  const toggleFavorite = async () => {
    try {
      await invoke(TAURI_COMMAND.TOGGLE_CLIPBOARD_ITEM_FAVORITE, {
        id: item.id,
      });

      onFavoriteToggled(item.id, !isFavorite);
    } catch (error) {
      log.error("toggle clipboard item favorite failed", error);
      message.error("操作失败");
    }
  };

  const remove = async () => {
    try {
      await invoke(TAURI_COMMAND.DELETE_CLIPBOARD_ITEM, { id: item.id });

      onRemoved(item.id);
    } catch (error) {
      log.error("delete clipboard item failed", error);
      message.error("删除失败");
    }
  };

  const confirmRemove = () => {
    if (!settingsState.clipboard.content.deleteConfirm) {
      remove();
      return;
    }

    modal.confirm({
      cancelText: "取消",
      content: "删除后无法恢复，确定删除这条记录吗？",
      okButtonProps: { danger: true },
      okText: "删除",
      onOk: remove,
      title: "删除记录",
    });
  };

  /**
   * 右键时即时构建并弹出原生菜单：通用项（粘贴 / 复制 / 收藏 / 备注 / 删除）恒在，
   * 类型相关项（纯文本粘贴 / 打开链接 / 发邮件 / 文件管理器显示）按 kind / subKind 追加。
   */
  const handleContextMenu = async (event: MouseEvent) => {
    event.preventDefault();

    const typed: MenuItemOptions[] = [];

    if (subKind === "url") {
      typed.push({ action: () => openLink(false), text: "打开链接" });
    }

    if (subKind === "email") {
      typed.push({ action: () => openLink(true), text: "发送邮件" });
    }

    if (subKind === "path" || kind === "files") {
      typed.push({
        action: reveal,
        text: isMac ? "在访达中显示" : "在资源管理器中显示",
      });
    }

    const items: Array<MenuItemOptions | PredefinedMenuItemOptions> = [
      { action: () => paste(false), text: "粘贴" },
      ...(kind === "text"
        ? [{ action: () => paste(true), text: "粘贴为纯文本" }]
        : []),
      { action: copy, text: "复制" },
      ...(typed.length > 0 ? [SEPARATOR, ...typed] : []),
      SEPARATOR,
      { action: toggleFavorite, text: isFavorite ? "取消收藏" : "收藏" },
      { action: () => onEditNote(item), text: "编辑备注" },
      SEPARATOR,
      { action: confirmRemove, text: "删除" },
    ];

    const menu = await Menu.new({ items });

    await menu.popup();
  };

  return handleContextMenu;
};
