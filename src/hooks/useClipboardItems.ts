import { useCallback, useEffect, useRef, useState } from "react";
import { listClipboardItems } from "@/commands";
import { TAURI_EVENT } from "@/constants/events";
import { useTauriListen } from "@/hooks/useTauriListen";
import type {
  ClipboardItem,
  ClipboardItemPage,
  ClipboardItemQuery,
  ClipboardPageReadyPayload,
} from "@/types/clipboard";
import { isDev } from "@/utils/is";
import { log } from "@/utils/log";
import { settleClipboardOpen } from "@/utils/openLatency";

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
interface UseClipboardItemsOptions {
  /**
   * 隐藏期预热页是否可被采纳。列表离开顶部时返回 false ——
   * 换掉整页会打乱用户当前浏览位置，那种情况留给打开时的正常刷新。
   */
  canAdoptReadyPage?: () => boolean;
  /**
   * 采纳回调：调用方据此认定「隐藏期那次刷新已经完成」，
   * 打开窗口时不必再补一次 reload。
   */
  onReadyPageAdopted?: () => void;
}

export const useClipboardItems = (
  query: ClipboardItemQuery,
  options?: UseClipboardItemsOptions,
) => {
  const queryRef = useRef(query);
  const optionsRef = useRef(options);
  optionsRef.current = options;

  const requestTokenRef = useRef(0);
  const itemsRef = useRef(new Map<number, ClipboardItem>());
  const totalRef = useRef(0);
  /** 上一次提交到 state 的数据版本号；与后端返回一致时说明内容未变，可跳过 setState。 */
  const revisionRef = useRef(-1);
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

  /**
   * 采纳 Rust 隐藏期算好的整页：直接替换本地镜像，作废在途请求。
   * 数据版本号同步推进，随后打开窗口时那次刷新会因版本一致而退化为 no-op。
   */
  const adoptReadyPage = useCallback(
    (page: ClipboardItemPage) => {
      const nextTotal = Math.max(0, page.total);
      const nextItems = new Map<number, ClipboardItem>();
      page.list.forEach((item, offset) => {
        if (offset < nextTotal) nextItems.set(offset, item);
      });

      // 在途请求此刻已经过期：它们的 token 被作废，回来不会覆盖这份更新的整页。
      requestTokenRef.current += 1;
      resetLoadingRanges();

      viewRangeRef.current = {
        end: Math.max(PAGE_SIZE - 1, page.list.length - 1),
        start: 0,
      };

      commitItems(nextItems);
      commitTotal(nextTotal);
      commitLoadedInitial(true);
      revisionRef.current = page.revision;
      setLoading(false);
    },
    [commitItems, commitLoadedInitial, commitTotal, resetLoadingRanges],
  );

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

      const startedAt = isDev ? performance.now() : 0;

      try {
        const page = await listClipboardItems({
          ...queryRef.current,
          limit: range.end - range.start + 1,
          offset: range.start,
        });

        if (options.token !== requestTokenRef.current) return;

        const requestMs = isDev ? performance.now() - startedAt : 0;
        const nextTotal = Math.max(0, page.total);

        // 数据版本号一致 + 该区间已完整加载 + 总数未变 → 这一页与屏幕上完全相同，
        // 不必重建 Map、也不必 setState（打开窗口时常见：这次刷新其实没有新内容）。
        if (
          page.revision === revisionRef.current &&
          nextTotal === totalRef.current &&
          isRangeLoaded(itemsRef.current, range)
        ) {
          if (isDev) {
            log.debug("clipboard list ipc (no-op)", {
              requestMs: Math.round(requestMs * 10) / 10,
              revision: page.revision,
            });
          }
          commitLoadedInitial(true);
          settleClipboardOpen("noop");
          return;
        }

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
        revisionRef.current = page.revision;

        if (isDev) {
          log.debug("clipboard list ipc", {
            items: page.list.length,
            offset: range.start,
            requestMs: Math.round(requestMs * 10) / 10,
            revision: page.revision,
          });
        }
        settleClipboardOpen("reload");
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

  // 窗口不可见时 Rust 已经把最新的第一页算好并推过来：直接采纳，
  // 打开窗口的那一刻列表就是最新内容，省掉那次跨进程往返。
  useTauriListen<ClipboardPageReadyPayload>(
    TAURI_EVENT.CLIPBOARD_PAGE_READY,
    ({ payload }) => {
      const page = payload?.page;
      if (!page) return;
      if (!isSameQuery(payload.query, queryRef.current)) return;
      // 不比已提交的版本新：内容没变，采纳只会白跑一次重渲染。
      if (page.revision <= revisionRef.current) return;
      if (optionsRef.current?.canAdoptReadyPage?.() === false) return;

      adoptReadyPage(page);
      optionsRef.current?.onReadyPageAdopted?.();

      if (isDev) {
        log.debug("clipboard list prewarmed page adopted", {
          items: page.list.length,
          revision: page.revision,
        });
      }
    },
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

/**
 * 过滤条件指纹：只比较影响结果集的字段（忽略 limit/offset 这类视窗参数），
 * `undefined` / `null` / 空白关键字统一归一，避免前后端序列化差异导致误判。
 */
function normalizeQueryKey(query: ClipboardItemQuery) {
  return {
    favorite: query.favorite ?? null,
    group: query.group ?? null,
    groupId: query.groupId ?? null,
    keyword: query.keyword?.trim() || null,
    kind: query.kind ?? null,
    pinned: query.pinned ?? null,
    sort: query.sort ?? null,
  };
}

function isSameQuery(a: ClipboardItemQuery, b: ClipboardItemQuery) {
  const left = normalizeQueryKey(a);
  const right = normalizeQueryKey(b);

  return (Object.keys(left) as Array<keyof typeof left>).every((key) => {
    return left[key] === right[key];
  });
}

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
