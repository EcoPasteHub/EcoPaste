import { Empty, Spin } from "antd";
import type { TFunction } from "i18next";
import type {
  FC,
  MouseEvent as ReactMouseEvent,
  PointerEvent as ReactPointerEvent,
} from "react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type TopItemListProps,
  Virtuoso,
  type VirtuosoHandle,
} from "react-virtuoso";
import { useSnapshot } from "valtio";
import {
  deleteClipboardItem,
  openClipboardItemLink,
  pasteClipboardItem,
  revealClipboardItem,
  toggleClipboardItemFavorite,
  toggleClipboardItemPinned,
  writeToClipboard,
} from "@/commands";
import { TAURI_EVENT } from "@/constants/events";
import { buildItemActionLabels } from "@/constants/itemActions";
import { WINDOW_LABEL } from "@/constants/windows";
import { useClipboardItems } from "@/hooks/useClipboardItems";
import { useKeyboardEvent } from "@/hooks/useKeyboardEvent";
import { useTauriListen } from "@/hooks/useTauriListen";
import {
  addClipboardPending,
  clearAllClipboardPending,
  clearClipboardPendingGroup,
  clipboardPendingState,
  hasAnyClipboardPending,
  hasClipboardPendingForGroup,
} from "@/stores/clipboardPending";
import { clipboardStatsState } from "@/stores/clipboardStats";
import { clipboardViewState } from "@/stores/clipboardView";
import { settingsState } from "@/stores/settings";
import type {
  ClipboardAction,
  ClipboardGroup,
  ClipboardItem,
  ClipboardItemSort,
  ClipboardKind,
} from "@/types/clipboard";
import type { ItemAction } from "@/types/settings";
import { cn } from "@/utils/cn";
import { isMac } from "@/utils/is";
import type { WindowVisibilityPayload } from "../hooks/previewController";
import {
  isSpaceKey,
  useClipboardPreviewController,
} from "../hooks/useClipboardPreviewController";
import ClipboardCard from "./cards/ClipboardCard";
import NoteModal from "./NoteModal";

/** 前 10 项的快捷键：index 0-8 对应 1-9，index 9 对应 0 */
const KEY_HINTS = ["1", "2", "3", "4", "5", "6", "7", "8", "9", "0"];

interface ClipboardUpdatedPayload {
  cleanup?: number;
  deduplicated?: boolean;
  id?: string;
  imported?: boolean;
  kind?: ClipboardKind;
}

/**
 * 剪贴板历史列表：虚拟滚动 + 分类型卡片 + 滚动到底分页加载，
 * 跟随关键词（Header 已防抖）检索。
 */
const List: FC = () => {
  const { t } = useTranslation("clipboard");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [firstVisibleIndex, setFirstVisibleIndex] = useState(0);
  const [isModifierPressed, setIsModifierPressed] = useState(false);
  const [noteTarget, setNoteTarget] = useState<ClipboardItem | null>(null);
  const virtuosoRef = useRef<VirtuosoHandle>(null);
  const isAtTopRef = useRef(true);
  const itemElementMapRef = useRef(new Map<string, HTMLDivElement>());
  const closePreviewRef = useRef<(reason: string) => void>(() => {});
  const displaySettingsMountedRef = useRef(false);
  const keywordRef = useRef("");
  const reloadRef = useRef<() => void>(() => {});

  const snapshot = useSnapshot(clipboardViewState);
  const pendingSnapshot = useSnapshot(clipboardPendingState);
  const settings = useSnapshot(settingsState);
  const { keyword, group } = snapshot;
  const { refreshGroup, refreshToken } = pendingSnapshot;
  const autoPaste = settings.clipboard.content.autoPaste;
  const middleClick = settings.clipboard.content.middleClick;
  const display = settings.clipboard.display;
  const sort = settings.clipboard.content.sort;
  const redactSecrets = settings.clipboard.sensitive.redactSecrets;
  const quickActions = settings.clipboard.content.itemActions;
  const deleteFavoriteItems = settings.clipboard.content.deleteFavoriteItems;
  const deletePinnedItems = settings.clipboard.content.deletePinnedItems;
  const deleteFavoriteItemsOnlyInFavoriteGroup =
    settings.clipboard.content.deleteFavoriteItemsOnlyInFavoriteGroup;
  const { fileMaxCount } = display;
  const showNewBadge = display.showNewBadge;
  const showOriginalPreview = settings.clipboard.content.showOriginalPreview;
  const quickActionLabels = buildItemActionLabels(t);

  const { data, loading, loadingMore, loadMore, noMore, reload, mutate } =
    useClipboardItems({ ...snapshot, sort });
  const items = data?.list ?? [];
  const topItemCount = countLeadingPinnedItems(items);
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
  closePreviewRef.current = closePreview;
  reloadRef.current = reload;

  // 把 Rust 返回的同过滤下总数同步给 Footer（共享 store），避免 Footer 单独 IPC 计数。
  useEffect(() => {
    if (data?.total !== void 0) clipboardStatsState.total = data.total;
  }, [data?.total]);

  useEffect(() => {
    if (showNewBadge) return;

    clearAllClipboardPending();
  }, [showNewBadge]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: snapshot 作触发器，不需在回调内读取
  useEffect(() => {
    setSelectedId(null);
    if (keywordRef.current !== keyword) {
      keywordRef.current = keyword;
      clearAllClipboardPending();
    }
    closePreview("filterChange");
  }, [snapshot]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: refreshToken 是 Group 点击 Badge 后的刷新信号
  useEffect(() => {
    if (refreshToken === 0) return;
    if (refreshGroup !== group) return;

    closePreview("showPending");
    setSelectedId(null);
    reloadRef.current();
    virtuosoRef.current?.scrollToIndex({ behavior: "smooth", index: 0 });
  }, [refreshToken, group, refreshGroup]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: 仅按影响列表 payload 的展示设置触发重拉，函数引用用 ref 读取最新值
  useEffect(() => {
    if (!displaySettingsMountedRef.current) {
      displaySettingsMountedRef.current = true;
      return;
    }

    closePreviewRef.current("displaySettingChange");
    reloadRef.current();
  }, [fileMaxCount, redactSecrets]);

  /**
   * 收到剪贴板更新：导入 / 清理强制刷新；普通新记录只在命中当前分组时刷新，
   * 其它分组的新内容累计到对应 Badge。
   * 用 ref 读取最新滚动位置，规避闭包陷旧值（事件订阅只挂载一次）。
   */
  const handleClipboardUpdated = (payload: ClipboardUpdatedPayload) => {
    if (payload.cleanup !== void 0) {
      closePreview("cleanup");
      setSelectedId(null);
      clearAllClipboardPending();
      if (clipboardStatsState.total !== null) {
        clipboardStatsState.total = Math.max(
          clipboardStatsState.total - payload.cleanup,
          0,
        );
      }
      virtuosoRef.current?.scrollToIndex({ behavior: "auto", index: 0 });
      reload();
      return;
    }

    if (payload.imported) {
      closePreview("backupImport");
      setSelectedId(null);
      clearAllClipboardPending();
      virtuosoRef.current?.scrollToIndex({ behavior: "auto", index: 0 });
      reload();
      return;
    }

    if (payload.deduplicated) {
      if (
        isAtTopRef.current &&
        shouldRefreshCurrentGroup(clipboardViewState.group, payload.kind)
      ) {
        reload();
      }

      return;
    }

    if (!isAtTopRef.current) {
      if (!showNewBadge) return;
      if (payload.kind === void 0) return;

      addClipboardPending(payload.kind);
      return;
    }

    if (shouldRefreshCurrentGroup(clipboardViewState.group, payload.kind)) {
      reload();
      return;
    }

    if (!showNewBadge) return;
    if (payload.kind === void 0) return;

    addClipboardPending(payload.kind);
  };

  useTauriListen<ClipboardUpdatedPayload>(
    TAURI_EVENT.CLIPBOARD_UPDATED,
    (event) => {
      handleClipboardUpdated(event.payload);
    },
  );

  /**
   * 主窗口显示时按偏好重置分组与滚动位置；隐藏期间攒下的新记录先触发刷新。
   */
  const handleWindowVisibility = (event: {
    payload: WindowVisibilityPayload;
  }) => {
    const { label, visible } = event.payload;
    if (label !== WINDOW_LABEL.MAIN || !visible) return;

    const { scrollToTopOnOpen, selectAllGroupOnOpen } =
      settings.clipboard.window;
    if (!scrollToTopOnOpen && !selectAllGroupOnOpen) return;

    closePreview("windowOpenReset");

    if (selectAllGroupOnOpen) {
      clipboardViewState.group = "all";
    }

    if (hasAnyClipboardPending()) {
      clearAllClipboardPending();
      reload();
    }

    if (!scrollToTopOnOpen) return;

    setSelectedId(null);
    virtuosoRef.current?.scrollToIndex({ behavior: "auto", index: 0 });
  };

  useTauriListen<WindowVisibilityPayload>(
    TAURI_EVENT.WINDOW_VISIBILITY,
    handleWindowVisibility,
  );

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
      list: sortClipboardItemsForView(
        data.list.map((item) => {
          return item.id === id ? { ...item, ...patch } : item;
        }),
        sort,
      ),
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
   * 打开条目备注编辑框；若该条正在预览，先关闭预览避免窗口层级互相遮挡。
   */
  const handleOpenNote = (item: ClipboardItem, reason: string) => {
    if (previewSession?.itemId === item.id) closePreview(reason);

    setNoteTarget(item);
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
    const target = items.find((item) => {
      return item.id === id;
    });

    if (!target || !canDeleteItem(target)) return;

    if (previewSession?.itemId === id) closePreview("delete");

    const deleted = await deleteClipboardItem(
      id,
      target.isFavorite,
      target.isPinned,
    );

    if (!deleted) return;

    setSelectedId(getSelectedIdAfterDelete(items, selectedId, id));
    removeItem(id);
  };

  /**
   * 快捷键触发的收藏切换：读当前项的 isFavorite 计算下一态，
   * Rust 返回真实状态后走统一的 `handleFavoriteToggled`（favorite 分组内取消会移除）。
   */
  const handleShortcutToggleFavorite = async (id: string) => {
    const current = items.find((item) => {
      return item.id === id;
    });

    if (!current) return;

    const next = await toggleClipboardItemFavorite(id, !current.isFavorite);

    handleFavoriteToggled(id, next);
  };

  /**
   * 切换条目置顶态；后端排序规则是置顶恒前置，本地镜像同步后按同规则重排。
   */
  const handleTogglePinned = async (id: string) => {
    const current = items.find((item) => {
      return item.id === id;
    });

    if (!current) return;

    const next = await toggleClipboardItemPinned(id, !current.isPinned);

    patchItem(id, { isPinned: next });
  };

  /**
   * 按当前条目后端声明的可用动作执行“打开”：链接 / 邮箱 / 定位文件共用 Cmd/Ctrl+O。
   */
  const handleShortcutOpen = async (
    item: ClipboardItem,
    action: ClipboardAction,
  ) => {
    if (previewSession?.itemId === item.id) closePreview("shortcutOpen");

    switch (action) {
      case "openLink":
        await openClipboardItemLink(item.id, false);
        return;
      case "sendEmail":
        await openClipboardItemLink(item.id, true);
        return;
      case "revealInFinder":
      case "revealInExplorer":
        await revealClipboardItem(item.id);
        return;
      default:
        return;
    }
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
      case "togglePinned":
        handleTogglePinned(target.id);
        return;
      case "editNote":
        handleOpenNote(target, "editNote");
        return;
      case "delete":
        if (!canDeleteItem(target)) return;

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

    const eventModifierPressed = isMac ? event.metaKey : event.ctrlKey;

    setIsModifierPressed(eventModifierPressed);

    if (event.key === "Enter") {
      event.preventDefault();

      const activeItem = getActiveItem();

      if (!activeItem) return;

      closePreview("enterPaste");
      pasteClipboardItem(activeItem.id, eventModifierPressed);

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
      eventModifierPressed &&
      (event.key === "Backspace" || event.key === "Delete")
    ) {
      event.preventDefault();

      const activeItem = getActiveItem();

      if (!activeItem) return;

      handleShortcutDelete(activeItem.id);

      return;
    }

    if (
      eventModifierPressed &&
      event.key.toLowerCase() === "c" &&
      !shouldUseNativeCopy(event)
    ) {
      event.preventDefault();

      const activeItem = getActiveItem();

      if (!activeItem) return;

      if (previewSession?.itemId === activeItem.id)
        closePreview("shortcutCopy");

      writeToClipboard(activeItem.id, false);

      return;
    }

    if (eventModifierPressed && event.key.toLowerCase() === "o") {
      const activeItem = getActiveItem();

      if (!activeItem) return;

      const openAction = getOpenClipboardAction(activeItem.availableActions);

      if (!openAction) return;

      event.preventDefault();
      void handleShortcutOpen(activeItem, openAction);

      return;
    }

    if (eventModifierPressed && event.key.toLowerCase() === "d") {
      event.preventDefault();

      const activeItem = getActiveItem();

      if (!activeItem) return;

      handleShortcutToggleFavorite(activeItem.id);

      return;
    }

    if (eventModifierPressed && event.key.toLowerCase() === "t") {
      event.preventDefault();

      const activeItem = getActiveItem();

      if (!activeItem) return;

      handleTogglePinned(activeItem.id);

      return;
    }

    if (eventModifierPressed && event.key.toLowerCase() === "m") {
      event.preventDefault();

      const activeItem = getActiveItem();

      if (!activeItem) return;

      handleOpenNote(activeItem, "shortcutNote");

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

  const handleKeyUp = (event: KeyboardEvent) => {
    const eventModifierPressed = isMac ? event.metaKey : event.ctrlKey;

    setIsModifierPressed(eventModifierPressed);
  };

  useKeyboardEvent("keyup", handleKeyUp);

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

    if (!atTop) return;
    if (!hasClipboardPendingForGroup(group)) return;

    clearClipboardPendingGroup(group);
    reload();
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

  if (loading && items.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Spin />
      </div>
    );
  }

  if (items.length === 0) {
    const description = getEmptyDescription(t, keyword, group);

    return (
      <div
        className="flex flex-1 flex-col items-center justify-center"
        data-tauri-drag-region
      >
        <Empty description={description} image={Empty.PRESENTED_IMAGE_SIMPLE} />
      </div>
    );
  }

  return (
    <div
      className="relative flex-1 overflow-hidden"
      onPointerLeave={handlePreviewAreaPointerLeave}
      role="listbox"
    >
      <Virtuoso
        atTopStateChange={handleAtTopStateChange}
        components={{ Footer: renderFooter, TopItemList }}
        computeItemKey={computeItemKey}
        data={items}
        endReached={handleEndReached}
        itemContent={renderItemContent}
        rangeChanged={handleRangeChanged}
        ref={virtuosoRef}
        topItemCount={topItemCount}
      />

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

    const handleOpenLink = () => {
      closePreview("openLink");
      openClipboardItemLink(item.id, item.subKind === "email");
    };

    const handleEditNote = () => {
      handleOpenNote(item, "editNote");
    };

    const handleQuickAction = async (action: ItemAction) => {
      if (action === "delete" && !canDeleteItem(item)) return;

      switch (action) {
        case "paste":
          closePreview("quickPaste");
          await pasteClipboardItem(item.id, false);
          return;
        case "pastePlain":
          closePreview("quickPastePlain");
          await pasteClipboardItem(item.id, true);
          return;
        case "pastePath":
          closePreview("quickPastePath");
          await pasteClipboardItem(item.id, true);
          return;
        case "copy":
          if (previewSession?.itemId === item.id) closePreview("quickCopy");
          await writeToClipboard(item.id, false);
          return;
        case "copyPlain":
          if (previewSession?.itemId === item.id) {
            closePreview("quickCopyPlain");
          }
          await writeToClipboard(item.id, true);
          return;
        case "openLink":
          if (previewSession?.itemId === item.id) {
            closePreview("quickOpenLink");
          }
          await openClipboardItemLink(item.id, false);
          return;
        case "sendEmail":
          if (previewSession?.itemId === item.id) {
            closePreview("quickSendEmail");
          }
          await openClipboardItemLink(item.id, true);
          return;
        case "reveal":
          if (previewSession?.itemId === item.id) closePreview("quickReveal");
          await revealClipboardItem(item.id);
          return;
        case "note":
          handleEditNote();
          return;
        case "pinItem":
          await handleTogglePinned(item.id);
          return;
        case "star":
          await handleShortcutToggleFavorite(item.id);
          return;
        case "delete":
          await handleShortcutDelete(item.id);
          return;
      }
    };

    const handleMouseDown = (event: ReactMouseEvent<HTMLDivElement>) => {
      if (event.button !== 0) {
        if (event.button !== 1) return;

        event.preventDefault();

        if (middleClick === "singleClickPaste") {
          setSelectedId(item.id);
          closePreview("middleClickPaste");
          pasteClipboardItem(item.id, false);
          return;
        }

        if (middleClick === "singleClickPastePlain") {
          setSelectedId(item.id);
          closePreview("middleClickPastePlain");
          pasteClipboardItem(item.id, true);
          return;
        }

        if (middleClick === "singleClickCopy") {
          setSelectedId(item.id);
          closePreview("middleClickCopy");
          writeToClipboard(item.id, false);
          return;
        }

        if (middleClick === "singleClickCopyPlain") {
          setSelectedId(item.id);
          closePreview("middleClickCopyPlain");
          writeToClipboard(item.id, true);
        }

        return;
      }

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

    const handleAuxClick = (event: ReactMouseEvent<HTMLDivElement>) => {
      if (event.button !== 1) return;

      event.preventDefault();
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

    const availableActions = getAllowedClipboardActions(
      item.availableActions,
      item,
      canDeleteItem,
    );
    const visibleQuickActions = getAllowedItemActions(
      quickActions,
      item,
      canDeleteItem,
    );

    return (
      <div className={cn("px-3", { "pt-3": index !== 0 })}>
        <ClipboardCard
          availableActions={availableActions}
          hintKey={hintKey}
          isLinkActive={isModifierPressed}
          isSelected={
            selectedId === null ? index === 0 : item.id === selectedId
          }
          item={item}
          onAuxClick={handleAuxClick}
          onDoubleClick={handleDoubleClick}
          onMouseDown={handleMouseDown}
          onOpenLink={handleOpenLink}
          onPointerEnter={handlePointerEnter}
          onPointerLeave={handlePointerLeave}
          onPointerMove={handlePointerMove}
          onQuickAction={handleQuickAction}
          onQuickPaste={hintKey ? handleQuickPaste : void 0}
          quickActionLabels={quickActionLabels}
          quickActions={visibleQuickActions}
          rootRef={registerItemElement(item.id)}
          showOriginalOnHover={showOriginalPreview}
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

  /**
   * 判断当前条目是否允许删除：收藏 / 置顶条目分别受各自保护开关约束。
   */
  function canDeleteItem(item: ClipboardItem) {
    if (item.isPinned && !deletePinnedItems) return false;

    if (!item.isFavorite) return true;

    if (!deleteFavoriteItems) return false;

    if (!deleteFavoriteItemsOnlyInFavoriteGroup) return true;

    return group === "favorite";
  }
};

/**
 * 生成空列表文案，区分搜索、收藏分组和普通空历史。
 */
function getEmptyDescription(
  t: TFunction<"clipboard">,
  keyword: string,
  group: string,
) {
  const isSearching = keyword.length > 0;
  const isFavorite = group === "favorite";

  if (isSearching) {
    return isFavorite
      ? t("empty.searchFavorites", { keyword })
      : t("empty.searchHistory", { keyword });
  }

  return t(isFavorite ? "empty.favorites" : "empty.history");
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

/**
 * 按条目保护规则过滤右键菜单动作，避免受保护项出现删除入口。
 */
function getAllowedClipboardActions(
  actions: ClipboardAction[] | undefined,
  item: ClipboardItem,
  canDeleteItem: (item: ClipboardItem) => boolean,
) {
  if (canDeleteItem(item)) return actions;

  return actions?.filter((action) => {
    return action !== "delete";
  });
}

/**
 * 从后端声明的右键动作中取出可由 Cmd/Ctrl+O 触发的“打开”动作。
 */
function getOpenClipboardAction(actions: ClipboardAction[] | undefined) {
  const openActions: ClipboardAction[] = [
    "openLink",
    "sendEmail",
    "revealInFinder",
    "revealInExplorer",
  ];

  return actions?.find((action) => {
    return openActions.includes(action);
  });
}

/**
 * 按条目保护规则过滤悬停快捷动作，避免受保护项出现删除按钮。
 */
function getAllowedItemActions(
  actions: readonly ItemAction[],
  item: ClipboardItem,
  canDeleteItem: (item: ClipboardItem) => boolean,
) {
  if (canDeleteItem(item)) return [...actions];

  return actions.filter((action) => {
    return action !== "delete";
  });
}

/**
 * 删除当前 active 项后优先选后一项；删除末尾时回退到前一项。
 * 右键删除非 active 项时保留当前显式选中，避免意外跳选。
 */
function getSelectedIdAfterDelete(
  items: ClipboardItem[],
  selectedId: string | null,
  deletedId: string,
) {
  const deletedIndex = items.findIndex((item) => {
    return item.id === deletedId;
  });

  if (deletedIndex === -1) return selectedId;

  const activeId = selectedId ?? items[0]?.id ?? null;

  if (activeId !== deletedId) return selectedId;

  const nextItem = items[deletedIndex + 1] ?? items[deletedIndex - 1] ?? null;

  return nextItem?.id ?? null;
}

/**
 * 判断 Cmd/Ctrl+C 是否应交给浏览器原生复制，避免覆盖输入框或文本选区复制。
 */
function shouldUseNativeCopy(event: KeyboardEvent) {
  const target = event.target;
  if (target instanceof HTMLElement) {
    const tagName = target.tagName.toLowerCase();
    if (target.isContentEditable) return true;
    if (tagName === "input" || tagName === "textarea") return true;
  }

  const selection = window.getSelection();

  return Boolean(selection && !selection.isCollapsed);
}

const computeItemKey = (_: number, item: ClipboardItem) => {
  return item.id;
};

/**
 * Virtuoso 的置顶项会 sticky 覆盖滚动内容；这里补实底色避免下方条目透出。
 */
const TopItemList: FC<TopItemListProps> = (props) => {
  const { children, style } = props;

  return (
    <div className="relative z-10 bg-ant-container" style={style}>
      {children}
    </div>
  );
};

/**
 * 统计当前已加载页开头连续置顶条目数，供 Virtuoso sticky top items 使用。
 */
function countLeadingPinnedItems(items: ClipboardItem[]) {
  let count = 0;

  for (const item of items) {
    if (!item.isPinned) break;

    count += 1;
  }

  return count;
}

/**
 * 判断普通剪贴板更新是否会出现在当前分组列表中。
 */
function shouldRefreshCurrentGroup(
  group: ClipboardGroup,
  kind?: ClipboardKind,
) {
  if (group === "all") return true;
  if (group === "favorite") return false;
  if (kind === void 0) return false;

  return group === kind;
}

/**
 * 按 Rust 查询层同款规则重排本地镜像：置顶恒前置，组内按当前 sort 下降。
 */
function sortClipboardItemsForView(
  items: ClipboardItem[],
  sort: ClipboardItemSort,
) {
  return [...items].sort((left, right) => {
    if (left.isPinned !== right.isPinned) {
      return left.isPinned ? -1 : 1;
    }

    if (sort === "useCountDesc") {
      return right.useCount - left.useCount;
    }

    const key = sort === "createdAtDesc" ? "createdAt" : "updatedAt";

    return right[key].localeCompare(left[key]);
  });
}

export default List;
