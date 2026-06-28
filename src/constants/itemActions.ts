/* @unocss-include */
import type { TFunction } from "i18next";
import type { ClipboardAction, ClipboardItem } from "@/types/clipboard";
import type { ItemAction } from "@/types/settings";

type ClipboardTranslator = TFunction<"clipboard">;

interface ItemActionMeta {
  activeLabelKey?: string;
  danger?: boolean;
  icon: string;
  labelKey: string;
  successIcon?: string;
  successLabelKey?: string;
}

export type ItemActionLabels = Record<ItemAction, string> & {
  copySuccess: string;
  pinItemActive: string;
  starActive: string;
};

export interface ItemActionState {
  copied?: boolean;
  isFavorite?: boolean;
  isPinned?: boolean;
}

export interface ItemActionPresentation {
  danger: boolean;
  icon: string;
  label: string;
}

export const ITEM_ACTION_ORDER: ItemAction[] = [
  "paste",
  "pastePlain",
  "pastePath",
  "copy",
  "copyPlain",
  "openLink",
  "sendEmail",
  "reveal",
  "note",
  "star",
  "pinItem",
  "delete",
];

export const ITEM_ACTION_META: Record<ItemAction, ItemActionMeta> = {
  copy: {
    icon: "i-lucide:copy",
    labelKey: "quickActions.copy",
    successIcon: "i-lucide:circle-check",
    successLabelKey: "quickActions.copySuccess",
  },
  copyPlain: {
    icon: "i-lucide:copy-check",
    labelKey: "quickActions.copyPlain",
    successIcon: "i-lucide:circle-check",
    successLabelKey: "quickActions.copySuccess",
  },
  delete: {
    danger: true,
    icon: "i-lucide:trash",
    labelKey: "quickActions.delete",
  },
  note: {
    icon: "i-lucide:notebook-pen",
    labelKey: "quickActions.note",
  },
  openLink: {
    icon: "i-lucide:square-arrow-out-up-right",
    labelKey: "quickActions.openLink",
  },
  paste: {
    icon: "i-lucide:clipboard-paste",
    labelKey: "quickActions.paste",
  },
  pastePath: {
    icon: "i-lucide:file-symlink",
    labelKey: "quickActions.pastePath",
  },
  pastePlain: {
    icon: "i-lucide:clipboard-type",
    labelKey: "quickActions.pastePlain",
  },
  pinItem: {
    activeLabelKey: "quickActions.pinItemActive",
    icon: "i-ph:push-pin-bold -rotate-45",
    labelKey: "quickActions.pinItem",
  },
  reveal: {
    icon: "i-lucide:folder-open",
    labelKey: "quickActions.reveal",
  },
  sendEmail: {
    icon: "i-lucide:mail",
    labelKey: "quickActions.sendEmail",
  },
  star: {
    activeLabelKey: "quickActions.starActive",
    icon: "i-lucide:star",
    labelKey: "quickActions.star",
  },
};

export const ITEM_ACTION_OPTIONS = ITEM_ACTION_ORDER.map((action) => {
  return { value: action };
});

const ITEM_ACTION_KEYS = new Set<ItemAction>(ITEM_ACTION_ORDER);

/**
 * 判断未知字符串是否为剪贴板条目快捷动作。
 */
export function isItemAction(value: string): value is ItemAction {
  return ITEM_ACTION_KEYS.has(value as ItemAction);
}

/**
 * 从 clipboard 命名空间翻译快捷动作文案。
 */
export function translateItemActionLabel(
  t: ClipboardTranslator,
  action: ItemAction,
  state: ItemActionState = {},
) {
  const meta = ITEM_ACTION_META[action];

  if (state.copied && meta.successLabelKey) {
    return t(meta.successLabelKey);
  }

  if ((state.isFavorite || state.isPinned) && meta.activeLabelKey) {
    return t(meta.activeLabelKey);
  }

  return t(meta.labelKey);
}

/**
 * 生成剪贴板卡片 hover 快捷动作使用的完整文案表。
 */
export function buildItemActionLabels(t: ClipboardTranslator) {
  const labels = ITEM_ACTION_ORDER.reduce(
    (result, action) => {
      result[action] = translateItemActionLabel(t, action);

      return result;
    },
    {} as Record<ItemAction, string>,
  );

  return {
    ...labels,
    copySuccess: translateItemActionLabel(t, "copy", { copied: true }),
    pinItemActive: translateItemActionLabel(t, "pinItem", {
      isPinned: true,
    }),
    starActive: translateItemActionLabel(t, "star", { isFavorite: true }),
  };
}

/**
 * 解析快捷动作按钮当前状态下的展示信息。
 */
export function resolveItemActionPresentation(
  action: ItemAction,
  labels: ItemActionLabels,
  state: ItemActionState = {},
): ItemActionPresentation {
  return {
    danger: isDangerItemAction(action),
    icon: resolveItemActionIcon(action, state),
    label: resolveItemActionLabel(action, labels, state),
  };
}

/**
 * 根据动作当前状态解析展示文案。
 */
export function resolveItemActionLabel(
  action: ItemAction,
  labels: ItemActionLabels,
  state: ItemActionState = {},
) {
  if (state.copied && isCopyItemAction(action)) return labels.copySuccess;

  if (state.isFavorite && action === "star") return labels.starActive;

  if (state.isPinned && action === "pinItem") return labels.pinItemActive;

  return labels[action];
}

/**
 * 根据动作当前状态解析图标。
 */
export function resolveItemActionIcon(
  action: ItemAction,
  state: ItemActionState = {},
) {
  const meta = ITEM_ACTION_META[action];

  if (state.copied && meta.successIcon) return meta.successIcon;

  return meta.icon;
}

/**
 * 判断动作是否需要 danger 视觉样式。
 */
export function isDangerItemAction(action: ItemAction) {
  return Boolean(ITEM_ACTION_META[action].danger);
}

/**
 * 判断动作是否为写回剪贴板动作，可展示复制成功临时反馈。
 */
export function isCopyItemAction(action: ItemAction) {
  return action === "copy" || action === "copyPlain";
}

/**
 * 按当前条目的可用右键菜单动作过滤 hover 快捷动作。
 */
export function filterAvailableItemActions(
  actions: readonly ItemAction[],
  item: ClipboardItem,
) {
  return actions.filter((action) => {
    return isItemActionAvailable(action, item);
  });
}

/**
 * 判断某个快捷动作在当前条目上是否有意义。
 */
export function isItemActionAvailable(action: ItemAction, item: ClipboardItem) {
  switch (action) {
    case "copyPlain":
      return item.kind !== "image" && hasClipboardAction(item, "copy");
    case "openLink":
      return hasClipboardAction(item, "openLink");
    case "paste":
      return hasClipboardAction(item, "paste");
    case "pastePath":
      return hasClipboardAction(item, "pasteAsPath");
    case "pastePlain":
      return hasClipboardAction(item, "pasteAsPlainText");
    case "reveal":
      return (
        hasClipboardAction(item, "revealInFinder") ||
        hasClipboardAction(item, "revealInExplorer")
      );
    case "sendEmail":
      return hasClipboardAction(item, "sendEmail");
    case "copy":
      return hasClipboardAction(item, "copy");
    case "delete":
      return hasClipboardAction(item, "delete");
    case "note":
      return hasClipboardAction(item, "editNote");
    case "pinItem":
      return hasClipboardAction(item, "togglePinned");
    case "star":
      return hasClipboardAction(item, "toggleFavorite");
  }
}

/**
 * 判断 Rust 返回的右键菜单动作列表是否包含目标动作。
 */
function hasClipboardAction(item: ClipboardItem, action: ClipboardAction) {
  return item.availableActions?.includes(action) ?? false;
}
