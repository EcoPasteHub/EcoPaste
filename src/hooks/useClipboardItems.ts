import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { useInfiniteScroll, useMount, useUnmount } from "ahooks";
import { useRef } from "react";
import { TAURI_COMMAND } from "@/constants/commands";
import { TAURI_EVENT } from "@/constants/events";
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
 * 拉取剪贴板历史，分页累积 + 订阅 `clipboard://updated` 自动刷新。
 *
 * 返回 ahooks `useInfiniteScroll` 的标准结构（`data.list` / `loading` /
 * `loadingMore` / `loadMore` / `noMore` / `reload`），调用方按需取用。
 */
export const useClipboardItems = (query: ClipboardItemQuery) => {
  const result = useInfiniteScroll<ClipboardItemsPage>(
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

  const unlistenRef = useRef<(() => void) | null>(null);

  /**
   * 订阅 Rust 端 `clipboard://updated` 事件，触发列表 reload（回到第一页）。
   */
  const subscribe = async () => {
    try {
      unlistenRef.current = await listen(
        TAURI_EVENT.CLIPBOARD_UPDATED,
        result.reload,
      );
    } catch (err) {
      log.error("listen clipboard://updated failed", err);
    }
  };

  /**
   * 卸载时取消订阅，避免窗口销毁后残留监听。
   */
  const unsubscribe = () => {
    unlistenRef.current?.();
    unlistenRef.current = null;
  };

  useMount(subscribe);

  useUnmount(unsubscribe);

  return result;
};
