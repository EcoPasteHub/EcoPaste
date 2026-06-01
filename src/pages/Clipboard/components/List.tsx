import { useCreation } from "ahooks";
import { Empty, Spin } from "antd";
import type { FC } from "react";
import { useEffect, useRef, useState } from "react";
import { Virtuoso, type VirtuosoHandle } from "react-virtuoso";
import { useSnapshot } from "valtio";
import { useClipboardItems } from "@/hooks/useClipboardItems";
import { useKeyboardEvent } from "@/hooks/useKeyboardEvent";
import { clipboardViewState } from "@/stores/clipboardView";
import type { ClipboardItem, ClipboardItemQuery } from "@/types/clipboard";
import { cn } from "@/utils/cn";
import ClipboardCard from "./cards/ClipboardCard";

/**
 * 剪贴板历史列表：虚拟滚动 + 分类型卡片 + 滚动到底分页加载，
 * 跟随关键词（Header 已防抖）检索。
 */
const List: FC = () => {
  const [selectedIndex, setSelectedIndex] = useState(0);
  const virtuosoRef = useRef<VirtuosoHandle>(null);

  const snapshot = useSnapshot(clipboardViewState);
  const { keyword, ...rest } = snapshot;

  const query = useCreation<ClipboardItemQuery>(
    () => ({ ...rest, keyword: keyword || void 0 }),
    [snapshot],
  );

  const { data, loading, loadingMore, loadMore, noMore } =
    useClipboardItems(query);
  const items = data?.list ?? [];

  // biome-ignore lint/correctness/useExhaustiveDependencies: query 作触发器，不需在回调内读取
  useEffect(() => {
    setSelectedIndex(0);
  }, [query]);

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key !== "ArrowUp" && e.key !== "ArrowDown") return;
    if (items.length === 0) return;

    e.preventDefault();

    const next =
      e.key === "ArrowUp"
        ? Math.max(0, selectedIndex - 1)
        : Math.min(items.length - 1, selectedIndex + 1);

    setSelectedIndex(next);
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

    return (
      <div
        className="flex flex-1 flex-col items-center justify-center"
        data-tauri-drag-region
      >
        <Empty
          description={
            isSearching ? `未找到「${keyword}」相关记录` : "暂无剪贴板历史"
          }
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </div>
    );
  }

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
        increaseViewportBy={400}
        itemContent={renderItemContent}
        ref={virtuosoRef}
      />
    </div>
  );

  function renderItemContent(index: number, item: ClipboardItem) {
    const handleMouseEnter = () => setSelectedIndex(index);

    return (
      <div className={cn("px-3", { "pt-3": index !== 0 })}>
        <ClipboardCard
          isSelected={index === selectedIndex}
          item={item}
          onMouseEnter={handleMouseEnter}
        />
      </div>
    );
  }
};

const computeItemKey = (_: number, item: ClipboardItem) => item.id;

export default List;
