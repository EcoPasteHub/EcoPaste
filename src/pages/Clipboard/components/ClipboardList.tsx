import { invoke } from "@tauri-apps/api/core";
import { useEffect, useRef } from "react";
import { Virtuoso, type VirtuosoHandle } from "react-virtuoso";
import { useSnapshot } from "valtio";
import { TAURI_COMMAND } from "@/constants/commands";
import { WINDOW_LABEL } from "@/constants/windows";
import type { ClipboardViewTab } from "@/stores/clipboardView";
import { settingsState } from "@/stores/settings";
import { log } from "@/utils/log";

import { useClipboardApps } from "../hooks/useClipboardApps";
import { useClipboardItems } from "../hooks/useClipboardItems";
import { useListNavigation } from "../hooks/useListNavigation";
import ClipboardCard from "./cards/ClipboardCard";

interface Props {
  keyword?: string;
  tab?: ClipboardViewTab;
}

const ClipboardList = ({ keyword = "", tab }: Props) => {
  const { items, loadMore, actions } = useClipboardItems(keyword, tab);
  const apps = useClipboardApps(items);
  const virtuosoRef = useRef<VirtuosoHandle>(null);
  const { value: settings } = useSnapshot(settingsState);
  const autoPaste = settings?.clipboard.content.autoPaste ?? "doubleClick";

  const { selectedIndex, setSelectedIndex } = useListNavigation({
    count: items.length,
    onEnter: (idx) => {
      const item = items[idx];
      if (!item) return;
      actions.paste(item.id);
    },
    onEscape: () => {
      invoke(TAURI_COMMAND.HIDE_WINDOW, { label: WINDOW_LABEL.MAIN }).catch(
        (err) => log.error("hide_window failed", err),
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
        // biome-ignore lint/a11y/noStaticElementInteractions: 列表行选中/粘贴交互，键盘路径由 useListNavigation 全局处理
        // biome-ignore lint/a11y/useKeyWithClickEvents: 同上，键盘走全局 keydown，不在 item 上重复挂
        <div
          onClick={() => {
            setSelectedIndex(idx);
            if (autoPaste === "singleClick") actions.paste(item.id);
          }}
          onDoubleClick={() => {
            if (autoPaste === "doubleClick") actions.paste(item.id);
          }}
        >
          <ClipboardCard
            actions={actions}
            app={item.sourceAppId ? apps.get(item.sourceAppId) : undefined}
            isSelected={idx === selectedIndex}
            item={item}
            keyword={keyword}
          />
        </div>
      )}
      ref={virtuosoRef}
      style={{ height: "100%" }}
    />
  );
};

export default ClipboardList;
