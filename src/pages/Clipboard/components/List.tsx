import { invoke } from "@tauri-apps/api/core";
import { useCreation } from "ahooks";
import { Empty, Spin } from "antd";
import type { FC } from "react";
import { useEffect, useRef, useState } from "react";
import { Virtuoso, type VirtuosoHandle } from "react-virtuoso";
import { useSnapshot } from "valtio";
import { TAURI_COMMAND } from "@/constants/commands";
import { useClipboardItems } from "@/hooks/useClipboardItems";
import { useKeyboardEvent } from "@/hooks/useKeyboardEvent";
import { clipboardViewState } from "@/stores/clipboardView";
import type { ClipboardItem, ClipboardItemQuery } from "@/types/clipboard";
import { cn } from "@/utils/cn";
import ClipboardCard from "./cards/ClipboardCard";

/** 前 10 项的快捷键：index 0-8 对应 1-9，index 9 对应 0 */
const KEY_HINTS = ["1", "2", "3", "4", "5", "6", "7", "8", "9", "0"];

/**
 * 剪贴板历史列表：虚拟滚动 + 分类型卡片 + 滚动到底分页加载，
 * 跟随关键词（Header 已防抖）检索。
 */
const List: FC = () => {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [firstVisibleIndex, setFirstVisibleIndex] = useState(0);
  const virtuosoRef = useRef<VirtuosoHandle>(null);

  const snapshot = useSnapshot(clipboardViewState);
  const { keyword, group, ...rest } = snapshot;

  const query = useCreation<ClipboardItemQuery>(
    () => ({
      ...rest,
      favorite: group === "favorite" ? true : void 0,
      keyword: keyword || void 0,
      kind: group === "all" || group === "favorite" ? void 0 : group,
    }),
    [snapshot],
  );

  const { data, loading, loadingMore, loadMore, noMore } =
    useClipboardItems(query);
  const items = data?.list ?? [];

  // biome-ignore lint/correctness/useExhaustiveDependencies: query 作触发器，不需在回调内读取
  useEffect(() => {
    setSelectedId(null);
  }, [query]);

  const handleKeyDown = (e: KeyboardEvent) => {
    if (items.length === 0) return;

    if (e.key === "Enter") {
      e.preventDefault();

      const activeId = selectedId === null ? items[0].id : selectedId;

      invoke(TAURI_COMMAND.PASTE_CLIPBOARD_ITEM, {
        id: activeId,
        plain: false,
      });

      return;
    }

    if (e.key !== "ArrowUp" && e.key !== "ArrowDown") return;

    e.preventDefault();

    const currentIndex =
      selectedId === null
        ? 0
        : Math.max(
            0,
            items.findIndex((item) => item.id === selectedId),
          );

    const next =
      e.key === "ArrowUp"
        ? Math.max(0, currentIndex - 1)
        : Math.min(items.length - 1, currentIndex + 1);

    setSelectedId(items[next].id);
    virtuosoRef.current?.scrollIntoView({ behavior: "smooth", index: next });
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
    const isSearching = keyword.length > 0;
    const isFavorite = group === "favorite";

    let description: string;

    if (isSearching) {
      description = isFavorite
        ? `未找到「${keyword}」相关收藏`
        : `未找到「${keyword}」相关记录`;
    } else {
      description = isFavorite ? "暂无收藏记录" : "暂无剪贴板历史";
    }

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
  };

  const handleEndReached = () => {
    if (!noMore && !loadingMore) loadMore();
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
    <div className="flex-1 overflow-hidden">
      <Virtuoso
        components={{ Footer: renderFooter }}
        computeItemKey={computeItemKey}
        data={items}
        endReached={handleEndReached}
        itemContent={renderItemContent}
        rangeChanged={handleRangeChanged}
        ref={virtuosoRef}
      />
    </div>
  );

  function renderItemContent(index: number, item: ClipboardItem) {
    const handleMouseEnter = () => setSelectedId(item.id);

    const relativeIndex = index - firstVisibleIndex;
    const hintKey =
      relativeIndex >= 0 && relativeIndex < 10
        ? KEY_HINTS[relativeIndex]
        : void 0;

    const handleQuickPaste = () => {
      invoke(TAURI_COMMAND.PASTE_CLIPBOARD_ITEM, { id: item.id, plain: false });
    };

    return (
      <div className={cn("px-3", { "pt-3": index !== 0 })}>
        <ClipboardCard
          hintKey={hintKey}
          isSelected={
            selectedId === null ? index === 0 : item.id === selectedId
          }
          item={item}
          onMouseEnter={handleMouseEnter}
          onQuickPaste={hintKey ? handleQuickPaste : void 0}
        />
      </div>
    );
  }
};

const computeItemKey = (_: number, item: ClipboardItem) => item.id;

export default List;
