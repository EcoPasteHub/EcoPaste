import {
  Menu,
  type MenuItemOptions,
  type PredefinedMenuItemOptions,
} from "@tauri-apps/api/menu";
import { App } from "antd";
import type { MouseEvent } from "react";
import {
  deleteClipboardItem,
  openClipboardItemLink,
  pasteClipboardItem,
  revealClipboardItem,
  toggleClipboardItemFavorite,
  writeToClipboard,
} from "@/commands";
import { settingsState } from "@/stores/settings";
import type { ClipboardItem } from "@/types/clipboard";
import { isMac } from "@/utils/is";

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
 * 所有动作都走 `@/commands` 包装：粘贴 / 复制 / 收藏 / 删除 / 备注 由 Rust 同名命令完成；
 * 打开链接 / 发邮件 / 文件管理器显示也下沉到 Rust（按 id 查 content + opener 调用），
 * 前端无需再回查完整 content。
 */
export const useContextMenu = (props: UseContextMenuProps) => {
  const { item, onRemoved, onFavoriteToggled, onEditNote } = props;
  const { kind, subKind, isFavorite } = item;

  const { modal } = App.useApp();

  const pasteItem = (plain: boolean) => pasteClipboardItem(item.id, plain);

  const copy = () => writeToClipboard(item.id, false);

  const openLink = (mailto: boolean) => openClipboardItemLink(item.id, mailto);

  const reveal = () => revealClipboardItem(item.id);

  const toggleFavorite = async () => {
    const next = await toggleClipboardItemFavorite(item.id, !isFavorite);
    onFavoriteToggled(item.id, next);
  };

  const remove = async () => {
    await deleteClipboardItem(item.id);
    onRemoved(item.id);
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
      { action: () => pasteItem(false), text: "粘贴" },
      ...(kind === "text"
        ? [{ action: () => pasteItem(true), text: "粘贴为纯文本" }]
        : []),
      ...(kind === "files"
        ? [{ action: () => pasteItem(true), text: "粘贴为路径" }]
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
