// 主窗剪贴板视图：tab 选择 + 派生的查询过滤。
// 仅 UI 状态，进程内存即可——不需要持久化（重开窗回到「全部」最直觉）。

import { proxy } from "valtio";

export type ClipboardViewTab =
  | { kind: "all" }
  | { kind: "favorite" }
  | { kind: "group"; groupId: string };

interface ClipboardViewState {
  tab: ClipboardViewTab;
}

export const clipboardViewState = proxy<ClipboardViewState>({
  tab: { kind: "all" },
});

export const setClipboardTab = (tab: ClipboardViewTab) => {
  clipboardViewState.tab = tab;
};

// HeroUI Tabs 的 key 是字符串：约定 "all" / "favorite" / "group:<id>"。
export const tabToKey = (tab: ClipboardViewTab): string => {
  switch (tab.kind) {
    case "all":
      return "all";
    case "favorite":
      return "favorite";
    case "group":
      return `group:${tab.groupId}`;
  }
};

export const keyToTab = (key: string): ClipboardViewTab => {
  if (key === "favorite") return { kind: "favorite" };
  if (key.startsWith("group:")) {
    return { groupId: key.slice("group:".length), kind: "group" };
  }
  return { kind: "all" };
};

// 在给定的 tab key 序列里循环移动一格；越界回绕。组件持有完整 key 列表（含动态分组），
// 故循环逻辑放在调用方而非 store——store 不依赖 groups 数据。
export const cycleTabKey = (
  keys: string[],
  current: string,
  direction: "next" | "prev",
): string => {
  if (keys.length === 0) return current;
  const idx = keys.indexOf(current);
  if (idx === -1) return keys[0] ?? current;
  const delta = direction === "next" ? 1 : -1;
  const nextIdx = (idx + delta + keys.length) % keys.length;
  return keys[nextIdx] ?? current;
};
