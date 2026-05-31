import { invoke } from "@tauri-apps/api/core";
import { useEffect, useState } from "react";
import { TAURI_COMMAND } from "@/constants/commands";
import type { ClipboardGroup } from "@/types/clipboard";
import { log } from "@/utils/log";

/**
 * 一次性加载分组列表。当前阶段不提供分组 CRUD UI，因此不监听变更事件——
 * 后续加上 add/rename/delete 命令后，在这里订阅同源事件 refetch 即可。
 */
export const useClipboardGroups = (): ClipboardGroup[] => {
  const [groups, setGroups] = useState<ClipboardGroup[]>([]);

  useEffect(() => {
    let cancelled = false;
    invoke<ClipboardGroup[]>(TAURI_COMMAND.LIST_CLIPBOARD_GROUPS)
      .then((list) => {
        if (!cancelled) setGroups(list);
      })
      .catch((err) => log.error("list_clipboard_groups failed", err));
    return () => {
      cancelled = true;
    };
  }, []);

  return groups;
};
