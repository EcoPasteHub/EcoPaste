import type { AppearanceGroupTab, AppearanceGroupTabId } from "@/types/store";

export type ClipboardGroupId = "all" | AppearanceGroupTabId;

export const GROUP_LABEL_KEYS = {
  all: "clipboard.label.tab.all",
  emoji: "clipboard.label.tab.emoji",
  favorite: "clipboard.label.tab.favorite",
  files: "clipboard.label.tab.files",
  gif: "clipboard.label.tab.gif",
  image: "clipboard.label.tab.image",
  kaomoji: "clipboard.label.tab.kaomoji",
  symbol: "clipboard.label.tab.symbol",
  text: "clipboard.label.tab.text",
} as const satisfies Record<ClipboardGroupId, string>;

export const DEFAULT_APPEARANCE_GROUP_TABS: AppearanceGroupTab[] = [
  { id: "emoji", visible: true },
  { id: "kaomoji", visible: true },
  { id: "symbol", visible: true },
  { id: "gif", visible: true },
  { id: "text", visible: true },
  { id: "image", visible: true },
  { id: "files", visible: true },
  { id: "favorite", visible: true },
];

export const normalizeAppearanceGroupTabs = (
  tabs?: AppearanceGroupTab[],
): AppearanceGroupTab[] => {
  const nextTabs = tabs ?? [];
  const visited = new Set<AppearanceGroupTab["id"]>();
  const normalizedTabs: AppearanceGroupTab[] = [];

  for (const currentTab of nextTabs) {
    const defaultTab = DEFAULT_APPEARANCE_GROUP_TABS.find(
      (item) => item.id === currentTab.id,
    );

    if (!defaultTab || visited.has(currentTab.id)) continue;

    normalizedTabs.push({
      id: currentTab.id,
      visible: currentTab.visible,
    });
    visited.add(currentTab.id);
  }

  for (const defaultTab of DEFAULT_APPEARANCE_GROUP_TABS) {
    if (visited.has(defaultTab.id)) continue;

    normalizedTabs.push({
      id: defaultTab.id,
      visible: defaultTab.visible,
    });
    visited.add(defaultTab.id);
  }

  return normalizedTabs;
};

export const getVisibleClipboardGroups = (tabs?: AppearanceGroupTab[]) => {
  return [
    "all",
    ...normalizeAppearanceGroupTabs(tabs)
      .filter((item) => item.visible)
      .map((item) => item.id),
  ] as ClipboardGroupId[];
};
