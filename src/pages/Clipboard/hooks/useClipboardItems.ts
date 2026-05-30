import { invoke } from "@tauri-apps/api/core";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { TAURI_COMMAND } from "@/constants/commands";
import { TAURI_EVENT } from "@/constants/events";
import { useTauriListen } from "@/hooks/useTauriListen";
import type { ClipboardViewTab } from "@/stores/clipboardView";
import type {
  ClipboardItem,
  ClipboardItemQuery,
  ClipboardItemSort,
} from "@/types/clipboard";
import { log } from "@/utils/log";

const PAGE_SIZE = 50;
// 关键词输入防抖：避免每个键入字符都打 Rust。
const KEYWORD_DEBOUNCE_MS = 200;

interface UpdatedPayload {
  id: string;
  deduplicated: boolean;
}

export interface ClipboardActions {
  // 写回剪贴板；不关闭窗口，不模拟粘贴。
  copy: (id: string, plain?: boolean) => Promise<void>;
  // 写回 + 隐藏窗口 + 模拟粘贴，Rust 侧 50ms 让位前台。
  paste: (id: string, plain?: boolean) => Promise<void>;
  toggleFavorite: (id: string) => Promise<void>;
  remove: (id: string) => Promise<void>;
  // 空串 / 全空白会被 Rust 端归一化为 NULL，等价于清空。
  updateNote: (id: string, note: string) => Promise<void>;
}

interface UseClipboardItemsResult {
  items: ClipboardItem[];
  loadMore: () => void;
  hasMore: boolean;
  actions: ClipboardActions;
}

const list = (query: ClipboardItemQuery) =>
  invoke<ClipboardItem[]>(TAURI_COMMAND.LIST_CLIPBOARD_ITEMS, { query });

const getOne = (id: string) =>
  invoke<ClipboardItem | null>(TAURI_COMMAND.GET_CLIPBOARD_ITEM, { id });

const tabToFilter = (
  tab: ClipboardViewTab,
): Pick<ClipboardItemQuery, "favorite" | "groupId"> => {
  switch (tab.kind) {
    case "favorite":
      return { favorite: true };
    case "group":
      return { groupId: tab.groupId };
    case "all":
      return {};
  }
};

const buildQuery = (
  keyword: string,
  tab: ClipboardViewTab,
  sort: ClipboardItemSort,
  limit: number,
  offset: number,
): ClipboardItemQuery => {
  const trimmed = keyword.trim();
  const base: ClipboardItemQuery = {
    limit,
    offset,
    sort,
    ...tabToFilter(tab),
  };
  return trimmed.length > 0 ? { ...base, keyword: trimmed } : base;
};

// 判定单条新/更新项是否属于当前视图——决定收到 `clipboard://updated` 事件后是否插入列表。
// 「全部」恒为真；「收藏」需 isFavorite；「分组」需 groupId 匹配。
const matchesTab = (item: ClipboardItem, tab: ClipboardViewTab): boolean => {
  switch (tab.kind) {
    case "all":
      return true;
    case "favorite":
      return item.isFavorite;
    case "group":
      return item.groupId === tab.groupId;
  }
};

export const useClipboardItems = (
  keyword: string = "",
  tab: ClipboardViewTab = { kind: "all" },
  sort: ClipboardItemSort = "createdAtDesc",
): UseClipboardItemsResult => {
  const [items, setItems] = useState<ClipboardItem[]>([]);
  const [hasMore, setHasMore] = useState(true);
  const itemsRef = useRef<ClipboardItem[]>(items);
  itemsRef.current = items;
  const loadingRef = useRef(false);
  // 当前生效的关键词（已防抖），loadMore / 事件回调读它，避免闭包陷旧值。
  const activeKeywordRef = useRef("");
  // 当前视图同理：tab 切换后立即刷新，事件回调据此判定是否插入。
  const tabRef = useRef(tab);
  tabRef.current = tab;
  const sortRef = useRef(sort);
  sortRef.current = sort;

  useEffect(() => {
    // 防抖期内 keyword 又变 → clearTimeout 撤掉，请求根本不发。
    // 已发请求期间 keyword/tab/sort 又变 → cancelled 拦住 setItems，避免旧结果覆盖新结果。
    let cancelled = false;
    const timer = window.setTimeout(() => {
      activeKeywordRef.current = keyword;
      loadingRef.current = true;
      list(buildQuery(keyword, tab, sort, PAGE_SIZE, 0))
        .then((page) => {
          if (cancelled) return;
          setItems(page);
          setHasMore(page.length === PAGE_SIZE);
        })
        .catch((err) => log.error("list_clipboard_items query failed", err))
        .finally(() => {
          if (!cancelled) loadingRef.current = false;
        });
    }, KEYWORD_DEBOUNCE_MS);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [keyword, tab, sort]);

  const loadMore = useCallback(() => {
    if (loadingRef.current || !hasMore) return;
    loadingRef.current = true;
    const offset = itemsRef.current.length;
    list(
      buildQuery(
        activeKeywordRef.current,
        tabRef.current,
        sortRef.current,
        PAGE_SIZE,
        offset,
      ),
    )
      .then((page) => {
        setItems((prev) => [...prev, ...page]);
        setHasMore(page.length === PAGE_SIZE);
      })
      .catch((err) => log.error("list_clipboard_items loadMore failed", err))
      .finally(() => {
        loadingRef.current = false;
      });
  }, [hasMore]);

  useTauriListen<UpdatedPayload>(TAURI_EVENT.CLIPBOARD_UPDATED, (payload) => {
    // 搜索态下结果是关键词快照：新条目未必匹配，强行前置会污染结果，索性跳过。
    // 清空搜索时 effect 会重新拉取，最新条目自然回到顶部。
    if (activeKeywordRef.current.trim().length > 0) return;
    getOne(payload.id)
      .then((item) => {
        if (!item) return;
        // 非「全部」视图：新条目可能不属于当前过滤集合（如普通条目进收藏视图），不插入。
        if (!matchesTab(item, tabRef.current)) return;
        setItems((prev) => {
          // 去重场景：旧条目移到顶部（updatedAt 已更新）。新条目：直接前置。
          const without = prev.filter((it) => it.id !== item.id);
          return [item, ...without];
        });
      })
      .catch((err) => log.error("get_clipboard_item failed", err));
  });

  // 写操作走乐观更新：先动本地、命令失败再 log 出来；失败时不回滚，
  // 因为剪贴板/数据库操作出错本身就罕见，整页 refetch 反而打断滚动。
  // 后续如要 toast 提示，统一在这里挂。
  const actions = useMemo<ClipboardActions>(
    () => ({
      copy: (id, plain = false) =>
        invoke<void>(TAURI_COMMAND.WRITE_TO_CLIPBOARD, { id, plain }).catch(
          (err) => {
            log.error("write_to_clipboard failed", err);
          },
        ),
      paste: (id, plain = false) =>
        invoke<void>(TAURI_COMMAND.PASTE_CLIPBOARD_ITEM, { id, plain }).catch(
          (err) => {
            log.error("paste_clipboard_item failed", err);
          },
        ),
      remove: (id) => {
        setItems((prev) => prev.filter((it) => it.id !== id));
        return invoke<void>(TAURI_COMMAND.DELETE_CLIPBOARD_ITEM, { id }).catch(
          (err) => log.error("delete_clipboard_item failed", err),
        );
      },
      toggleFavorite: (id) => {
        // 在「收藏」视图下取消收藏后，该条不再属于当前过滤集合 → 从列表移除。
        // 在其他视图下仅翻转标记，保留位置。
        setItems((prev) => {
          const currentTab = tabRef.current;
          return prev.flatMap((it) => {
            if (it.id !== id) return [it];
            const next = { ...it, isFavorite: !it.isFavorite };
            if (currentTab.kind === "favorite" && !next.isFavorite) return [];
            return [next];
          });
        });
        return invoke<void>(TAURI_COMMAND.TOGGLE_CLIPBOARD_ITEM_FAVORITE, {
          id,
        }).catch((err) =>
          log.error("toggle_clipboard_item_favorite failed", err),
        );
      },
      updateNote: (id, note) => {
        const trimmed = note.trim();
        const next = trimmed.length === 0 ? null : trimmed;
        setItems((prev) =>
          prev.map((it) => (it.id === id ? { ...it, note: next } : it)),
        );
        return invoke<boolean>(TAURI_COMMAND.UPDATE_CLIPBOARD_ITEM_NOTE, {
          id,
          note: next,
        })
          .then((autoFavorited) => {
            if (!autoFavorited) return;
            setItems((prev) =>
              prev.map((it) =>
                it.id === id ? { ...it, isFavorite: true } : it,
              ),
            );
          })
          .catch((err) => log.error("update_clipboard_item_note failed", err));
      },
    }),
    [],
  );

  return { actions, hasMore, items, loadMore };
};
