import { invoke } from "@tauri-apps/api/core";
import { useEffect, useState } from "react";
import { Virtuoso } from "react-virtuoso";

import type { ClipboardItem } from "@/types/clipboard";
import { log } from "@/utils/log";

const Row = ({ item }: { item: ClipboardItem }) => (
  <div className="border-divider border-b px-3 py-2 text-sm">
    <div className="text-default-500 text-xs">{item.kind}</div>
    <div className="truncate">{item.content.slice(0, 80)}</div>
  </div>
);

const ClipboardList = () => {
  const [items, setItems] = useState<ClipboardItem[]>([]);

  useEffect(() => {
    invoke<ClipboardItem[]>("list_clipboard_items")
      .then(setItems)
      .catch((err) => log.error("list_clipboard_items failed", err));
  }, []);

  return (
    <Virtuoso
      computeItemKey={(_, item) => item.id}
      data={items}
      itemContent={(_, item) => <Row item={item} />}
      style={{ height: "100%" }}
    />
  );
};

export default ClipboardList;
