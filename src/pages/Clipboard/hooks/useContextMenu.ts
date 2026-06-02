import {
  Menu,
  type MenuItemOptions,
  type PredefinedMenuItemOptions,
} from "@tauri-apps/api/menu";
import type { MouseEvent } from "react";
import {
  deleteClipboardItem,
  openClipboardItemLink,
  pasteClipboardItem,
  revealClipboardItem,
  toggleClipboardItemFavorite,
  writeToClipboard,
} from "@/commands";
import type { ClipboardAction, ClipboardItem } from "@/types/clipboard";

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
 * 菜单视觉分组：组间插入分隔线，组内按 Rust 返回的 `availableActions` 顺序取交集。
 * 所有「这条记录能做哪些动作」的判定都在 Rust，前端只负责把动作映射成文案/快捷键/平台差异。
 */
const ACTION_GROUPS: ClipboardAction[][] = [
  ["paste", "pasteAsPlainText", "pasteAsPath", "copy"],
  ["openLink", "sendEmail", "revealInFinder", "revealInExplorer"],
  ["toggleFavorite", "editNote"],
  ["delete"],
];

/**
 * 列表项右键菜单：返回 `handleContextMenu`，绑到卡片根节点的 `onContextMenu`，
 * 右键时按 Rust 计算好的 `availableActions` 顺序拼 Tauri 原生菜单（`Menu.popup`）。
 */
export const useContextMenu = (props: UseContextMenuProps) => {
  const { item, onRemoved, onFavoriteToggled, onEditNote } = props;
  const { availableActions = [], isFavorite } = item;
  const actionSet = new Set<ClipboardAction>(availableActions);

  const pasteItem = (plain: boolean) => pasteClipboardItem(item.id, plain);

  const copy = () => writeToClipboard(item.id, false);

  const openLink = (mailto: boolean) => openClipboardItemLink(item.id, mailto);

  const reveal = () => revealClipboardItem(item.id);

  const toggleFavorite = async () => {
    const next = await toggleClipboardItemFavorite(item.id, !isFavorite);
    onFavoriteToggled(item.id, next);
  };

  const remove = async () => {
    const deleted = await deleteClipboardItem(item.id);

    if (deleted) onRemoved(item.id);
  };

  const buildMenuItem = (action: ClipboardAction): MenuItemOptions | null => {
    switch (action) {
      case "paste":
        return {
          accelerator: "Enter",
          action: () => pasteItem(false),
          text: "粘贴",
        };
      case "pasteAsPlainText":
        return {
          accelerator: "CmdOrCtrl+Enter",
          action: () => pasteItem(true),
          text: "粘贴为纯文本",
        };
      case "pasteAsPath":
        return {
          accelerator: "CmdOrCtrl+Enter",
          action: () => pasteItem(true),
          text: "粘贴为路径",
        };
      case "copy":
        return { action: copy, text: "复制" };
      case "openLink":
        return { action: () => openLink(false), text: "打开链接" };
      case "sendEmail":
        return { action: () => openLink(true), text: "发送邮件" };
      case "revealInFinder":
        return { action: reveal, text: "在访达中显示" };
      case "revealInExplorer":
        return { action: reveal, text: "在资源管理器中显示" };
      case "toggleFavorite":
        return {
          accelerator: "CmdOrCtrl+D",
          action: toggleFavorite,
          text: isFavorite ? "取消收藏" : "收藏",
        };
      case "editNote":
        return { action: () => onEditNote(item), text: "编辑备注" };
      case "delete":
        return {
          accelerator: "CmdOrCtrl+Backspace",
          action: remove,
          text: "删除",
        };
      default:
        return null;
    }
  };

  const handleContextMenu = async (event: MouseEvent) => {
    event.preventDefault();

    const items: Array<MenuItemOptions | PredefinedMenuItemOptions> = [];

    for (const group of ACTION_GROUPS) {
      const groupItems = group
        .filter((action) => actionSet.has(action))
        .map(buildMenuItem)
        .filter((entry): entry is MenuItemOptions => entry !== null);

      if (groupItems.length === 0) continue;

      if (items.length > 0) items.push(SEPARATOR);

      items.push(...groupItems);
    }

    if (items.length === 0) return;

    const menu = await Menu.new({ items });

    await menu.popup();
  };

  return handleContextMenu;
};
