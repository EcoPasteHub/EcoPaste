import { Empty, Spin } from "antd";
import type {
  FC,
  MouseEvent as ReactMouseEvent,
  PointerEvent as ReactPointerEvent,
} from "react";
import { useEffect, useRef, useState } from "react";
import { Virtuoso, type VirtuosoHandle } from "react-virtuoso";
import { useSnapshot } from "valtio";
import {
  deleteClipboardItem,
  openClipboardItemLink,
  pasteClipboardItem,
  revealClipboardItem,
  toggleClipboardItemFavorite,
  writeToClipboard,
} from "@/commands";
import { TAURI_EVENT } from "@/constants/events";
import { useClipboardItems } from "@/hooks/useClipboardItems";
import { useKeyboardEvent } from "@/hooks/useKeyboardEvent";
import { useTauriListen } from "@/hooks/useTauriListen";
import { clipboardStatsState } from "@/stores/clipboardStats";
import { clipboardViewState } from "@/stores/clipboardView";
import { settingsState } from "@/stores/settings";
import type { ClipboardAction, ClipboardItem } from "@/types/clipboard";
import { cn } from "@/utils/cn";
import { isMac } from "@/utils/is";
import {
  isSpaceKey,
  useClipboardPreviewController,
} from "../hooks/useClipboardPreviewController";
import ClipboardCard from "./cards/ClipboardCard";
import NoteModal from "./NoteModal";

/** 前 10 项的快捷键：index 0-8 对应 1-9，index 9 对应 0 */
const KEY_HINTS = ["1", "2", "3", "4", "5", "6", "7", "8", "9", "0"];

/**
 * 剪贴板历史列表：虚拟滚动 + 分类型卡片 + 滚动到底分页加载，
 * 跟随关键词（Header 已防抖）检索。
 */
const List: FC = () => {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [firstVisibleIndex, setFirstVisibleIndex] = useState(0);
  const [isAtTop, setIsAtTop] = useState(true);
  const [pendingCount, setPendingCount] = useState(0);
  const [noteTarget, setNoteTarget] = useState<ClipboardItem | null>(null);
  const virtuosoRef = useRef<VirtuosoHandle>(null);
  const isAtTopRef = useRef(true);
  const itemElementMapRef = useRef(new Map<string, HTMLDivElement>());

  const snapshot = useSnapshot(clipboardViewState);
  const settings = useSnapshot(settingsState);
  const { keyword, group } = snapshot;
  const autoPaste = settings.clipboard.content.autoPaste;

  const { data, loading, loadingMore, loadMore, noMore, reload, mutate } =
    useClipboardItems(snapshot);
  const items = data?.list ?? [];
  const {
    closeHoverPreviewForScroll,
    closePreview,
    handleItemPointerEnter,
    handleItemPointerLeave,
    handleItemPointerMove,
    handleKeyboardPreviewMove,
    handlePreviewAreaPointerLeave,
    handlePreviewSpaceDown,
    previewSession,
  } = useClipboardPreviewController({
    getActiveItem,
    itemElementMapRef,
    onHoverSelect: setSelectedId,
  });

  // 把 Rust 返回的同过滤下总数同步给 Footer（共享 store），避免 Footer 单独 IPC 计数。
  useEffect(() => {
    if (data?.total !== void 0) clipboardStatsState.total = data.total;
  }, [data?.total]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: snapshot 作触发器，不需在回调内读取
  useEffect(() => {
    setSelectedId(null);
    setPendingCount(0);
    closePreview("filterChange");
  }, [snapshot]);

  /**
   * 收到剪贴板更新：用户在顶部时直接刷新；否则累加待刷新计数，避免 Virtuoso 抖动。
   * 用 ref 读取最新滚动位置，规避闭包陷旧值（事件订阅只挂载一次）。
   */
  const handleClipboardUpdated = () => {
    if (isAtTopRef.current) {
      reload();
      return;
    }

    // 非顶部不 reload（避免 Virtuoso 抖动），但底部「共 N 项」要立即体现新增，
    // 否则总数会停在最近一次拉取的快照上，直到用户点「N 条新记录」才刷新。
    if (clipboardStatsState.total !== null) clipboardStatsState.total += 1;
    setPendingCount((n) => n + 1);
  };

  useTauriListen(TAURI_EVENT.CLIPBOARD_UPDATED, handleClipboardUpdated);

  /**
   * 删除 / 收藏 / 备注命令均不广播 clipboard://updated，故就地改本地镜像，
   * 避免整页 reload 打断滚动与选中态。
   */
  const removeItem = (id: string) => {
    if (!data) return;

    mutate({ ...data, list: data.list.filter((item) => item.id !== id) });
  };

  const patchItem = (id: string, patch: Partial<ClipboardItem>) => {
    if (!data) return;

    mutate({
      ...data,
      list: data.list.map((item) => {
        return item.id === id ? { ...item, ...patch } : item;
      }),
    });
  };

  /**
   * 收藏切换后：favorite 分组下取消收藏的条目应即时移出列表，其余分组仅更新标记。
   */
  const handleFavoriteToggled = (id: string, isFavorite: boolean) => {
    if (group === "favorite" && !isFavorite) {
      removeItem(id);
      return;
    }

    patchItem(id, { isFavorite });
  };

  /**
   * 备注保存后同步本地镜像；后端可能因 autoFavorite 设置联动收藏，故一并回填。
   */
  const handleNoteSaved = (
    id: string,
    note: string | null,
    autoFavorited: boolean,
  ) => {
    patchItem(id, autoFavorited ? { isFavorite: true, note } : { note });
  };

  const handleCloseNote = () => {
    setNoteTarget(null);
  };

  /**
   * 读取当前选中项。无显式选中时，列表第一项即键盘 active item。
   */
  function getActiveItem() {
    if (items.length === 0) return null;

    if (selectedId === null) return items[0];

    return (
      items.find((item) => {
        return item.id === selectedId;
      }) ?? null
    );
  }

  /**
   * 注册虚拟列表项对应的 DOM 节点，预览打开时用它采集 anchor rect。
   */
  const registerItemElement = (id: string) => {
    return (node: HTMLDivElement | null) => {
      if (node) {
        itemElementMapRef.current.set(id, node);
        return;
      }

      itemElementMapRef.current.delete(id);
    };
  };

  /**
   * 快捷键触发的删除：复用 `deleteClipboardItem` 内置的二次确认弹窗，
   * 仅当用户确认且 Rust 删除成功时才同步本地镜像。
   */
  const handleShortcutDelete = async (id: string) => {
    if (previewSession?.itemId === id) closePreview("delete");

    const deleted = await deleteClipboardItem(id);

    if (!deleted) return;

    removeItem(id);
  };

  /**
   * 快捷键触发的收藏切换：读当前项的 isFavorite 计算下一态，
   * Rust 返回真实状态后走统一的 `handleFavoriteToggled`（favorite 分组内取消会移除）。
   */
  const handleShortcutToggleFavorite = async (id: string) => {
    const current = items.find((item) => item.id === id);

    if (!current) return;

    const next = await toggleClipboardItemFavorite(id, !current.isFavorite);

    handleFavoriteToggled(id, next);
  };

  /**
   * Rust 右键菜单点击事件：携带 `{action, itemId}`。
   * 用 ref 持续指向「当前 render 的派发函数」，规避 `useTauriListen` 只在挂载时
   * 抓一次闭包导致的状态过期（同款做法见 `handleClipboardUpdated`）。
   */
  const handleMenuActionRef = useRef<
    (payload: { action: ClipboardAction; itemId: string }) => void
  >(() => {});
  handleMenuActionRef.current = (payload) => {
    const { action, itemId } = payload;
    const target = items.find((entry) => entry.id === itemId);

    if (!target) return;

    switch (action) {
      case "paste":
        closePreview("paste");
        pasteClipboardItem(target.id, false);
        return;
      case "pasteAsPlainText":
      case "pasteAsPath":
        closePreview("pastePlain");
        pasteClipboardItem(target.id, true);
        return;
      case "copy":
        if (previewSession?.itemId === target.id) closePreview("copy");
        writeToClipboard(target.id, false);
        return;
      case "openLink":
        if (previewSession?.itemId === target.id) closePreview("openLink");
        openClipboardItemLink(target.id, false);
        return;
      case "sendEmail":
        if (previewSession?.itemId === target.id) closePreview("sendEmail");
        openClipboardItemLink(target.id, true);
        return;
      case "revealInFinder":
      case "revealInExplorer":
        if (previewSession?.itemId === target.id) closePreview("reveal");
        revealClipboardItem(target.id);
        return;
      case "toggleFavorite":
        handleShortcutToggleFavorite(target.id);
        return;
      case "editNote":
        if (previewSession?.itemId === target.id) closePreview("editNote");
        setNoteTarget(target);
        return;
      case "delete":
        handleShortcutDelete(target.id);
        return;
    }
  };

  const handleMenuActionEvent = (event: { payload: unknown }) => {
    handleMenuActionRef.current(
      event.payload as { action: ClipboardAction; itemId: string },
    );
  };

  useTauriListen(TAURI_EVENT.CLIPBOARD_MENU_ACTION, handleMenuActionEvent);

  const handleKeyDown = (event: KeyboardEvent) => {
    if (items.length === 0) return;

    const isModifierPressed = isMac ? event.metaKey : event.ctrlKey;

    if (event.key === "Enter") {
      event.preventDefault();

      const activeItem = getActiveItem();

      if (!activeItem) return;

      closePreview("enterPaste");
      pasteClipboardItem(activeItem.id, isModifierPressed);

      return;
    }

    if (isSpaceKey(event)) {
      handlePreviewSpaceDown(event);
      return;
    }

    if (event.key === "Escape" && previewSession !== null) {
      event.preventDefault();
      closePreview("escape");

      return;
    }

    if (
      isModifierPressed &&
      (event.key === "Backspace" || event.key === "Delete")
    ) {
      event.preventDefault();

      const activeItem = getActiveItem();

      if (!activeItem) return;

      handleShortcutDelete(activeItem.id);

      return;
    }

    if (isModifierPressed && event.key.toLowerCase() === "d") {
      event.preventDefault();

      const activeItem = getActiveItem();

      if (!activeItem) return;

      handleShortcutToggleFavorite(activeItem.id);

      return;
    }

    if (event.key !== "ArrowUp" && event.key !== "ArrowDown") return;

    event.preventDefault();

    const next = getNextKeyboardTarget(event);

    if (!next) return;

    setSelectedId(next.item.id);
    virtuosoRef.current?.scrollIntoView({
      behavior: "smooth",
      index: next.index,
    });

    handleKeyboardPreviewMove(next.item);
  };

  useKeyboardEvent("keydown", handleKeyDown);

  if (loading && items.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Spin />
      </div>
    );
  }

  if (items.length === 0) {
    const description = getEmptyDescription(keyword, group);

    return (
      <div
        className="flex flex-1 flex-col items-center justify-center"
        data-tauri-drag-region
      >
        <Empty description={description} image={Empty.PRESENTED_IMAGE_SIMPLE} />
      </div>
    );
  }

  const handleRangeChanged = ({
    startIndex,
  }: {
    startIndex: number;
    endIndex: number;
  }) => {
    setFirstVisibleIndex(startIndex);

    closeHoverPreviewForScroll();
  };

  const handleEndReached = () => {
    if (!noMore && !loadingMore) loadMore();
  };

  const handleAtTopStateChange = (atTop: boolean) => {
    isAtTopRef.current = atTop;
    setIsAtTop(atTop);

    if (atTop && pendingCount > 0) {
      setPendingCount(0);
      reload();
    }
  };

  const handleShowPending = () => {
    closePreview("showPending");
    setPendingCount(0);
    reload();
    virtuosoRef.current?.scrollToIndex({ behavior: "smooth", index: 0 });
  };

  const renderFooter = () => {
    if (loadingMore) {
      return (
        <div className="flex justify-center py-2">
          <Spin size="small" />
        </div>
      );
    }

    return null;
  };

  return (
    <div
      className="relative flex-1 overflow-hidden"
      onPointerLeave={handlePreviewAreaPointerLeave}
      role="listbox"
    >
      <Virtuoso
        atTopStateChange={handleAtTopStateChange}
        components={{ Footer: renderFooter }}
        computeItemKey={computeItemKey}
        data={items}
        endReached={handleEndReached}
        itemContent={renderItemContent}
        rangeChanged={handleRangeChanged}
        ref={virtuosoRef}
      />

      {pendingCount > 0 && !isAtTop && (
        <button
          className="absolute top-2 left-1/2 z-10 -translate-x-1/2 rounded-full bg-ant-primary px-3 py-1 text-ant-light-solid text-xs shadow-md transition-opacity hover:opacity-90"
          onClick={handleShowPending}
          type="button"
        >
          {pendingCount} 条新记录，点击查看
        </button>
      )}

      <NoteModal
        item={noteTarget}
        onClose={handleCloseNote}
        onSaved={handleNoteSaved}
      />
    </div>
  );

  function renderItemContent(index: number, item: ClipboardItem) {
    const handlePointerEnter = (event: ReactPointerEvent<HTMLDivElement>) => {
      handleItemPointerEnter(item, event);
    };

    const handlePointerLeave = () => {
      handleItemPointerLeave();
    };

    const handlePointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
      handleItemPointerMove(item, event);
    };

    const relativeIndex = index - firstVisibleIndex;
    const hintKey =
      relativeIndex >= 0 && relativeIndex < 10
        ? KEY_HINTS[relativeIndex]
        : void 0;

    const handleQuickPaste = () => {
      closePreview("quickPaste");
      pasteClipboardItem(item.id, false);
    };

    const handleMouseDown = (event: ReactMouseEvent<HTMLDivElement>) => {
      if (event.button !== 0) return;

      setSelectedId(item.id);

      if (autoPaste === "singleClickPaste") {
        closePreview("singleClickPaste");
        pasteClipboardItem(item.id, false);
        return;
      }

      if (autoPaste === "singleClickCopy") {
        closePreview("singleClickCopy");
        writeToClipboard(item.id, false);
      }
    };

    const handleDoubleClick = () => {
      if (autoPaste === "doubleClickPaste") {
        closePreview("doubleClickPaste");
        pasteClipboardItem(item.id, false);
        return;
      }

      if (autoPaste === "doubleClickCopy") {
        closePreview("doubleClickCopy");
        writeToClipboard(item.id, false);
      }
    };

    return (
      <div className={cn("px-3", { "pt-3": index !== 0 })}>
        <ClipboardCard
          hintKey={hintKey}
          isSelected={
            selectedId === null ? index === 0 : item.id === selectedId
          }
          item={item}
          onDoubleClick={handleDoubleClick}
          onMouseDown={handleMouseDown}
          onPointerEnter={handlePointerEnter}
          onPointerLeave={handlePointerLeave}
          onPointerMove={handlePointerMove}
          onQuickPaste={hintKey ? handleQuickPaste : void 0}
          rootRef={registerItemElement(item.id)}
        />
      </div>
    );
  }

  /**
   * 根据方向键计算下一项。
   */
  function getNextKeyboardTarget(event: KeyboardEvent) {
    const nextIndex = getNextKeyboardIndex(items, selectedId, event.key);

    return { index: nextIndex, item: items[nextIndex] };
  }
};

/**
 * 生成空列表文案，区分搜索、收藏分组和普通空历史。
 */
function getEmptyDescription(keyword: string, group: string) {
  const isSearching = keyword.length > 0;
  const isFavorite = group === "favorite";

  if (isSearching) {
    return isFavorite
      ? `未找到「${keyword}」相关收藏`
      : `未找到「${keyword}」相关记录`;
  }

  return isFavorite ? "暂无收藏记录" : "暂无剪贴板历史";
}

/**
 * 根据方向键和当前选中 id 计算下一项索引。
 */
function getNextKeyboardIndex(
  items: ClipboardItem[],
  selectedId: string | null,
  key: string,
) {
  const currentIndex =
    selectedId === null
      ? 0
      : Math.max(
          0,
          items.findIndex((item) => {
            return item.id === selectedId;
          }),
        );

  if (key === "ArrowUp") {
    return Math.max(0, currentIndex - 1);
  }

  return Math.min(items.length - 1, currentIndex + 1);
}

const computeItemKey = (_: number, item: ClipboardItem) => {
  return item.id;
};

export default List;
