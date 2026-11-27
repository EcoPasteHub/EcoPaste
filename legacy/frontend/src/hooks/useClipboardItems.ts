import { useCallback, useEffect, useRef, useState } from "react";
import { listClipboardItems } from "@/commands";
import type { ClipboardItem, ClipboardItemQuery } from "@/types/clipboard";

/**
 * 后端分页大小。保持略大于一屏的原有取值，range cache 以此对齐请求边界。
 */
const PAGE_SIZE = 30;
const PRELOAD_ROWS = 30;
const CACHE_MAX_ROWS = 180;
const CACHE_KEEP_RADIUS = 90;

interface ClipboardItemsRange {
  end: number;
  start: number;
}

interface FetchRangeOptions {
  force?: boolean;
  replace?: boolean;
  token: number;
}

/**
 * 剪贴板列表 range cache：Rust 仍是查询、排序、搜索和 payload 裁剪的唯一真相；
 * 前端只按 Virtuoso 的可视范围缓存少量已加载行，避免无限滚动后持有完整列表。
 */
export const useClipboardItems = (query: ClipboardItemQuery) => {
  const queryRef = useRef(query);

  const requestTokenRef = useRef(0);
  const itemsRef = useRef(new Map<number, ClipboardItem>());
  const totalRef = useRef(0);
  const loadingRangesRef = useRef<ClipboardItemsRange[]>([]);
  const loadedInitialRef = useRef(false);
  const viewRangeRef = useRef<ClipboardItemsRange>({
    end: PAGE_SIZE - 1,
    start: 0,
  });

  const [items, setItems] = useState(() => new Map<number, ClipboardItem>());
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadedInitial, setLoadedInitial] = useState(false);
  const [loadingRangeCount, setLoadingRangeCount] = useState(0);

  const commitItems = useCallback((nextItems: Map<number, ClipboardItem>) => {
    itemsRef.current = nextItems;
    setItems(nextItems);
  }, []);

  const commitTotal = useCallback((nextTotal: number) => {
    totalRef.current = nextTotal;
    setTotal(nextTotal);
  }, []);

  const commitLoadedInitial = useCallback((nextLoadedInitial: boolean) => {
    loadedInitialRef.current = nextLoadedInitial;
    setLoadedInitial(nextLoadedInitial);
  }, []);

  const resetLoadingRanges = useCallback(() => {
    loadingRangesRef.current = [];
    setLoadingRangeCount(0);
  }, []);

  const addLoadingRange = useCallback((range: ClipboardItemsRange) => {
    loadingRangesRef.current = [...loadingRangesRef.current, range];
    setLoadingRangeCount(loadingRangesRef.current.length);
  }, []);

  const removeLoadingRange = useCallback((range: ClipboardItemsRange) => {
    loadingRangesRef.current = loadingRangesRef.current.filter((current) => {
      return current.start !== range.start || current.end !== range.end;
    });
    setLoadingRangeCount(loadingRangesRef.current.length);
  }, []);

  const fetchRange = useCallback(
    async (
      rawStartIndex: number,
      rawEndIndex: number,
      options: FetchRangeOptions,
    ) => {
      const range = normalizeFetchRange(
        rawStartIndex,
        rawEndIndex,
        totalRef.current,
      );
      if (range === null) return;

      const currentItems = itemsRef.current;
      if (!options.force) {
        if (isRangeLoaded(currentItems, range)) return;
        if (hasCoveringLoadingRange(loadingRangesRef.current, range)) return;
      }

      addLoadingRange(range);

      try {
        const page = await listClipboardItems({
          ...queryRef.current,
          limit: range.end - range.start + 1,
          offset: range.start,
        });

        if (options.token !== requestTokenRef.current) return;

        const nextTotal = Math.max(0, page.total);
        const nextItems = options.replace
          ? new Map<number, ClipboardItem>()
          : new Map(itemsRef.current);

        page.list.forEach((item, offset) => {
          const index = range.start + offset;
          if (index < nextTotal) nextItems.set(index, item);
        });

        trimCache(nextItems, viewRangeRef.current, nextTotal);
        commitItems(nextItems);
        commitTotal(nextTotal);
        commitLoadedInitial(true);
      } catch {
        // 命令包装层已统一 log + toast；这里只避免初始请求失败后卡在 loading。
        if (
          options.token === requestTokenRef.current &&
          !loadedInitialRef.current
        ) {
          commitItems(new Map());
          commitTotal(0);
          commitLoadedInitial(true);
        }
      } finally {
        if (options.token === requestTokenRef.current) {
          removeLoadingRange(range);
          setLoading(false);
        }
      }
    },
    [
      addLoadingRange,
      commitItems,
      commitLoadedInitial,
      commitTotal,
      removeLoadingRange,
    ],
  );

  const reload = useCallback(() => {
    const token = requestTokenRef.current + 1;
    requestTokenRef.current = token;
    resetLoadingRanges();
    if (!loadedInitialRef.current) {
      commitItems(new Map());
      commitTotal(0);
      commitLoadedInitial(false);
      setLoading(true);
    }
    viewRangeRef.current = {
      end: PAGE_SIZE - 1,
      start: 0,
    };

    void fetchRange(0, PAGE_SIZE - 1, {
      force: true,
      replace: true,
      token,
    });
  }, [
    commitItems,
    commitLoadedInitial,
    commitTotal,
    fetchRange,
    resetLoadingRanges,
  ]);

  const resetAndReload = useCallback(() => {
    const token = requestTokenRef.current + 1;
    requestTokenRef.current = token;
    resetLoadingRanges();
    commitItems(new Map());
    commitTotal(0);
    commitLoadedInitial(false);
    setLoading(true);
    viewRangeRef.current = {
      end: PAGE_SIZE - 1,
      start: 0,
    };

    void fetchRange(0, PAGE_SIZE - 1, {
      force: true,
      replace: true,
      token,
    });
  }, [
    commitItems,
    commitLoadedInitial,
    commitTotal,
    fetchRange,
    resetLoadingRanges,
  ]);

  const reloadCurrentRange = useCallback(() => {
    const token = requestTokenRef.current + 1;
    requestTokenRef.current = token;
    resetLoadingRanges();
    commitItems(new Map());

    const { end, start } = viewRangeRef.current;
    void fetchRange(start - PRELOAD_ROWS, end + PRELOAD_ROWS, {
      force: true,
      replace: true,
      token,
    });
  }, [commitItems, fetchRange, resetLoadingRanges]);

  const loadRange = useCallback(
    (startIndex: number, endIndex: number) => {
      viewRangeRef.current = {
        end: Math.max(startIndex, endIndex),
        start: Math.max(0, Math.min(startIndex, endIndex)),
      };

      void fetchRange(startIndex - PRELOAD_ROWS, endIndex + PRELOAD_ROWS, {
        token: requestTokenRef.current,
      });
    },
    [fetchRange],
  );

  const getItem = useCallback(
    (index: number) => {
      return items.get(index) ?? null;
    },
    [items],
  );

  const findItemById = useCallback(
    (id: string) => {
      for (const item of items.values()) {
        if (item.id === id) return item;
      }

      return null;
    },
    [items],
  );

  const getItemIndexById = useCallback(
    (id: string) => {
      for (const [index, item] of items) {
        if (item.id === id) return index;
      }

      return null;
    },
    [items],
  );

  const removeItemById = useCallback(
    (id: string) => {
      const removeIndex = getItemIndexById(id);
      if (removeIndex === null) return;

      const nextItems = new Map<number, ClipboardItem>();
      for (const [index, item] of itemsRef.current) {
        if (item.id === id) continue;

        nextItems.set(index > removeIndex ? index - 1 : index, item);
      }

      const nextTotal = Math.max(0, totalRef.current - 1);
      trimCache(nextItems, viewRangeRef.current, nextTotal);
      commitItems(nextItems);
      commitTotal(nextTotal);
      void fetchRange(
        viewRangeRef.current.start - PRELOAD_ROWS,
        viewRangeRef.current.end + PRELOAD_ROWS,
        {
          force: true,
          token: requestTokenRef.current,
        },
      );
    },
    [commitItems, commitTotal, fetchRange, getItemIndexById],
  );

  const patchItemById = useCallback(
    (id: string, patch: Partial<ClipboardItem>) => {
      const index = getItemIndexById(id);
      if (index === null) return;

      const current = itemsRef.current.get(index);
      if (!current) return;

      const nextItems = new Map(itemsRef.current);
      nextItems.set(index, { ...current, ...patch });
      commitItems(nextItems);
    },
    [commitItems, getItemIndexById],
  );

  useEffect(() => {
    queryRef.current = {
      favorite: query.favorite,
      group: query.group,
      groupId: query.groupId,
      keyword: query.keyword,
      kind: query.kind,
      pinned: query.pinned,
      sort: query.sort,
    };
    resetAndReload();
  }, [
    resetAndReload,
    query.favorite,
    query.group,
    query.groupId,
    query.keyword,
    query.kind,
    query.pinned,
    query.sort,
  ]);

  return {
    findItemById,
    getItem,
    getItemIndexById,
    loadedInitial,
    loading,
    loadingMore: loadingRangeCount > 0 && loadedInitial,
    loadRange,
    patchItemById,
    reload,
    reloadCurrentRange,
    removeItemById,
    total,
  };
};

function normalizeFetchRange(
  rawStartIndex: number,
  rawEndIndex: number,
  total: number,
): ClipboardItemsRange | null {
  if (rawEndIndex < 0) return null;

  const maxKnownIndex = total > 0 ? total - 1 : Math.max(0, rawEndIndex);
  const clampedStart = Math.max(0, Math.min(rawStartIndex, maxKnownIndex));
  const clampedEnd = Math.max(
    clampedStart,
    Math.min(rawEndIndex, maxKnownIndex),
  );
  const start = Math.floor(clampedStart / PAGE_SIZE) * PAGE_SIZE;
  const end = Math.ceil((clampedEnd + 1) / PAGE_SIZE) * PAGE_SIZE - 1;

  return {
    end: total > 0 ? Math.min(end, total - 1) : end,
    start,
  };
}

function isRangeLoaded(
  items: Map<number, ClipboardItem>,
  range: ClipboardItemsRange,
) {
  for (let index = range.start; index <= range.end; index += 1) {
    if (!items.has(index)) return false;
  }

  return true;
}

function hasCoveringLoadingRange(
  ranges: ClipboardItemsRange[],
  target: ClipboardItemsRange,
) {
  return ranges.some((range) => {
    return range.start <= target.start && range.end >= target.end;
  });
}

function trimCache(
  items: Map<number, ClipboardItem>,
  viewRange: ClipboardItemsRange,
  total: number,
) {
  if (items.size <= CACHE_MAX_ROWS) return;

  const center = Math.floor((viewRange.start + viewRange.end) / 2);
  const keepStart = Math.max(0, center - CACHE_KEEP_RADIUS);
  const keepEnd = Math.min(total - 1, center + CACHE_KEEP_RADIUS);
  const leadingPinnedEnd = getLeadingPinnedEnd(items);

  for (const [index] of items) {
    if (index <= leadingPinnedEnd) continue;
    if (index < keepStart || index > keepEnd) items.delete(index);
  }
}

function getLeadingPinnedEnd(items: Map<number, ClipboardItem>) {
  let index = 0;
  let lastPinnedIndex = -1;

  while (true) {
    const item = items.get(index);
    if (!item?.isPinned) return lastPinnedIndex;

    lastPinnedIndex = index;
    index += 1;
  }
}
