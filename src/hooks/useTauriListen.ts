import {
  type EventCallback,
  type EventName,
  listen,
} from "@tauri-apps/api/event";
import { useMount, useUnmount } from "ahooks";
import { useRef } from "react";
import { log } from "@/utils/log";

/**
 * 订阅 Tauri 事件，组件卸载时自动取消监听。
 * 等价于 `listen(event, handler)` + cleanup，消除样板代码。
 */
export const useTauriListen = <T>(
  event: EventName,
  handler: EventCallback<T>,
) => {
  const unlistenRef = useRef<(() => void) | undefined>(void 0);

  useMount(async () => {
    try {
      unlistenRef.current = await listen<T>(event, handler);
    } catch (error) {
      log.error(`Failed to listen tauri event: ${event}`, error);
    }
  });

  useUnmount(() => {
    unlistenRef.current?.();
  });
};
