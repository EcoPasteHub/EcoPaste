import { invoke } from "@tauri-apps/api/core";
import { useInfiniteScroll } from "ahooks";
import { TAURI_COMMAND } from "@/constants/commands";
import type { ClipboardItem, ClipboardItemQuery } from "@/types/clipboard";
import { log } from "@/utils/log";

/**
 * 单页大小。略大于一屏可见数，滚到底再触发下一页，体验更顺。
 */
const PAGE_SIZE = 30;

interface ClipboardItemsPage {
  list: ClipboardItem[];
  /** 后端无总数接口，按本页条数 == 页大小近似判断是否还有下一页。 */
  hasMore: boolean;
}

/**
 * 拉取剪贴板历史，分页累积。
 *
 * 返回 ahooks `useInfiniteScroll` 的标准结构（`data.list` / `loading` /
 * `loadingMore` / `loadMore` / `noMore` / `reload`），调用方按需取用。
 *
 * 注意：`clipboard://updated` 事件**不**在此 hook 里直接 `reload`——否则用户
 * 滚到中段时插入新数据会导致 Virtuoso 抖动。由 `List` 组件按「是否在顶部」
 * 决定立即刷新还是延后提示。
 */
export const useClipboardItems = (query: ClipboardItemQuery) => {
  return useInfiniteScroll<ClipboardItemsPage>(
    async (current) => {
      const offset = current?.list.length ?? 0;
      const list = await invoke<ClipboardItem[]>(
        TAURI_COMMAND.LIST_CLIPBOARD_ITEMS,
        { query: { ...query, limit: PAGE_SIZE, offset } },
      );

      return { hasMore: list.length === PAGE_SIZE, list };
    },
    {
      isNoMore: (data) => data?.hasMore === false,
      onError: (err) => log.error("list_clipboard_items failed", err),
      reloadDeps: [query],
    },
  );
};
