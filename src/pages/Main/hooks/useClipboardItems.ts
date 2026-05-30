import { invoke } from "@tauri-apps/api/core";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useTauriListen } from "@/hooks/useTauriListen";
import type { ClipboardItem, ClipboardItemQuery } from "@/types/clipboard";
import { log } from "@/utils/log";

const PAGE_SIZE = 50;
const CLIPBOARD_UPDATED = "clipboard://updated";

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
  invoke<ClipboardItem[]>("list_clipboard_items", { query });

const getOne = (id: string) =>
  invoke<ClipboardItem | null>("get_clipboard_item", { id });

export const useClipboardItems = (): UseClipboardItemsResult => {
  const [items, setItems] = useState<ClipboardItem[]>([]);
  const [hasMore, setHasMore] = useState(true);
  const itemsRef = useRef<ClipboardItem[]>(items);
  itemsRef.current = items;
  const loadingRef = useRef(false);

  useEffect(() => {
    list({ limit: PAGE_SIZE, offset: 0 })
      .then((page) => {
        setItems(page);
        setHasMore(page.length === PAGE_SIZE);
      })
      .catch((err) => log.error("list_clipboard_items initial failed", err));
  }, []);

  const loadMore = useCallback(() => {
    if (loadingRef.current || !hasMore) return;
    loadingRef.current = true;
    const offset = itemsRef.current.length;
    list({ limit: PAGE_SIZE, offset })
      .then((page) => {
        setItems((prev) => [...prev, ...page]);
        setHasMore(page.length === PAGE_SIZE);
      })
      .catch((err) => log.error("list_clipboard_items loadMore failed", err))
      .finally(() => {
        loadingRef.current = false;
      });
  }, [hasMore]);

  useTauriListen<UpdatedPayload>(CLIPBOARD_UPDATED, (payload) => {
    getOne(payload.id)
      .then((item) => {
        if (!item) return;
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
        invoke<void>("write_to_clipboard", { id, plain }).catch((err) => {
          log.error("write_to_clipboard failed", err);
        }),
      paste: (id, plain = false) =>
        invoke<void>("paste_clipboard_item", { id, plain }).catch((err) => {
          log.error("paste_clipboard_item failed", err);
        }),
      remove: (id) => {
        setItems((prev) => prev.filter((it) => it.id !== id));
        return invoke<void>("delete_clipboard_item", { id }).catch((err) =>
          log.error("delete_clipboard_item failed", err),
        );
      },
      toggleFavorite: (id) => {
        setItems((prev) =>
          prev.map((it) =>
            it.id === id ? { ...it, isFavorite: !it.isFavorite } : it,
          ),
        );
        return invoke<void>("toggle_clipboard_item_favorite", { id }).catch(
          (err) => log.error("toggle_clipboard_item_favorite failed", err),
        );
      },
      updateNote: (id, note) => {
        const trimmed = note.trim();
        const next = trimmed.length === 0 ? null : trimmed;
        setItems((prev) =>
          prev.map((it) => (it.id === id ? { ...it, note: next } : it)),
        );
        return invoke<void>("update_clipboard_item_note", {
          id,
          note: next,
        }).catch((err) => log.error("update_clipboard_item_note failed", err));
      },
    }),
    [],
  );

  return { actions, hasMore, items, loadMore };
};
