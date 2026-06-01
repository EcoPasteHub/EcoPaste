import { invoke } from "@tauri-apps/api/core";
import { useMount } from "ahooks";
import { useCallback, useState } from "react";
import { TAURI_COMMAND } from "@/constants/commands";
import { TAURI_EVENT } from "@/constants/events";
import { log } from "@/utils/log";
import { useTauriListen } from "./useTauriListen";

/**
 * 订阅剪贴板历史总条数，初始拉取一次后跟随 `clipboard://updated` 事件自动刷新。
 */
export const useClipboardTotal = () => {
  const [total, setTotal] = useState(0);

  const fetchTotal = useCallback(async () => {
    try {
      const count = await invoke<number>(TAURI_COMMAND.COUNT_CLIPBOARD_ITEMS);

      setTotal(count);
    } catch (error) {
      log.error("count_clipboard_items failed", error);
    }
  }, []);

  useMount(fetchTotal);

  useTauriListen(TAURI_EVENT.CLIPBOARD_UPDATED, fetchTotal);

  return total;
};
