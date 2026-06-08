import { proxy } from "valtio";
import type { ClipboardGroup, ClipboardKind } from "@/types/clipboard";

type ClipboardPendingCounts = Record<ClipboardKind, number>;

interface ClipboardPendingState {
  counts: ClipboardPendingCounts;
  refreshGroup: ClipboardGroup | null;
  refreshToken: number;
}

/**
 * 分组按钮上的未查看新记录数，以及用户点击 Badge 后给列表消费的刷新信号。
 */
export const clipboardPendingState = proxy<ClipboardPendingState>({
  counts: {
    files: 0,
    image: 0,
    text: 0,
  },
  refreshGroup: null,
  refreshToken: 0,
});

/**
 * 按真实内容类型累计新记录。
 */
export const addClipboardPending = (kind: ClipboardKind) => {
  clipboardPendingState.counts[kind] += 1;
};

/**
 * 清除单个分组的 Badge 数字；全部分组覆盖所有类型。
 */
export const clearClipboardPendingGroup = (group: ClipboardGroup) => {
  if (group === "all") {
    clearAllClipboardPending();
    return;
  }

  if (group === "favorite") return;

  clipboardPendingState.counts[group] = 0;
};

/**
 * 清除全部分组 Badge 数字，供导入、清理、搜索过滤变更等全量刷新场景使用。
 */
export const clearAllClipboardPending = () => {
  clipboardPendingState.counts.files = 0;
  clipboardPendingState.counts.image = 0;
  clipboardPendingState.counts.text = 0;
};

/**
 * 用户点击带 Badge 的分组后发出刷新请求，List 消费 token 后滚动到顶部。
 */
export const requestClipboardPendingRefresh = (group: ClipboardGroup) => {
  clearClipboardPendingGroup(group);
  clipboardPendingState.refreshGroup = group;
  clipboardPendingState.refreshToken += 1;
};

/**
 * 判断目标分组是否有待查看内容；全部分组覆盖所有类型 Badge。
 */
export const hasClipboardPendingForGroup = (group: ClipboardGroup) => {
  if (group === "favorite") return false;
  if (group !== "all") return clipboardPendingState.counts[group] > 0;

  return hasAnyClipboardPending();
};

/**
 * 判断是否有任何待查看新记录。
 */
export const hasAnyClipboardPending = () => {
  return Object.values(clipboardPendingState.counts).some((count) => {
    return count > 0;
  });
};

/**
 * 获取分组 Badge 展示数量；全部分组展示所有待查看记录总数。
 */
export const getClipboardPendingCount = (group: ClipboardGroup) => {
  if (group === "favorite") return 0;
  if (group !== "all") return clipboardPendingState.counts[group];

  return Object.values(clipboardPendingState.counts).reduce((total, count) => {
    return total + count;
  }, 0);
};
