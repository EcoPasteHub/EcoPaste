import { invoke } from "@tauri-apps/api/core";
import { useEffect, useRef } from "react";
import { Virtuoso, type VirtuosoHandle } from "react-virtuoso";

import { log } from "@/utils/log";

import { useClipboardItems } from "../hooks/useClipboardItems";
import { useListNavigation } from "../hooks/useListNavigation";
import ClipboardCard from "./cards/ClipboardCard";

const ClipboardList = () => {
  const { items, loadMore, actions } = useClipboardItems();
  const virtuosoRef = useRef<VirtuosoHandle>(null);

  const { selectedIndex } = useListNavigation({
    count: items.length,
    onEnter: (idx) => {
      const item = items[idx];
      if (!item) return;
      actions.paste(item.id);
    },
    onEscape: () => {
      invoke("hide_window", { label: "main" }).catch((err) =>
        log.error("hide_window failed", err),
      );
    },
  });

  useEffect(() => {
    virtuosoRef.current?.scrollToIndex({ index: selectedIndex });
  }, [selectedIndex]);

  return (
    <Virtuoso
      computeItemKey={(_, item) => item.id}
      data={items}
      endReached={loadMore}
      increaseViewportBy={200}
      itemContent={(idx, item) => (
        <ClipboardCard
          actions={actions}
          isSelected={idx === selectedIndex}
          item={item}
        />
      )}
      ref={virtuosoRef}
      style={{ height: "100%" }}
    />
  );
};

export default ClipboardList;
