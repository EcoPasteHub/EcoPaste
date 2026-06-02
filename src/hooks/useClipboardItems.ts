import { useInfiniteScroll } from "ahooks";
import { listClipboardItems } from "@/commands";
import type { ClipboardItem, ClipboardItemQuery } from "@/types/clipboard";

/**
 * 单页大小。略大于一屏可见数，滚到底再触发下一页，体验更顺。
 */
const PAGE_SIZE = 30;

interface ClipboardItemsPage {
  list: ClipboardItem[];
  /** 同过滤下的总条数，Rust 返回，供 Footer「共 N 项」共用，避免单独发 IPC。 */
  total: number;
  /** Rust 用 `offset + list.len() < total` 精确判定，前端不再用 `len === pageSize` 近似。 */
  hasMore: boolean;
}

/**
 * 拉取剪贴板历史，分页累积。
 *
 * 返回 ahooks `useInfiniteScroll` 的标准结构（`data.list` / `data.total` /
 * `loading` / `loadingMore` / `loadMore` / `noMore` / `reload`），调用方按需取用。
 *
 * 注意：`clipboard://updated` 事件**不**在此 hook 里直接 `reload`——否则用户
 * 滚到中段时插入新数据会导致 Virtuoso 抖动。由 `List` 组件按「是否在顶部」
 * 决定立即刷新还是延后提示。
 */
export const useClipboardItems = (query: ClipboardItemQuery) => {
  return useInfiniteScroll<ClipboardItemsPage>(
    async (current) => {
      const offset = current?.list.length ?? 0;
      const page = await listClipboardItems({
        ...query,
        limit: PAGE_SIZE,
        offset,
      });

      return {
        hasMore: page.hasMore,
        list: page.list,
        total: page.total,
      };
    },
    {
      isNoMore: (data) => data?.hasMore === false,
      reloadDeps: [query],
    },
  );
};
