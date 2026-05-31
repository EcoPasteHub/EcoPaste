import { useMount, useUnmount } from "@reactuses/core";
import { listen, type UnlistenFn } from "@tauri-apps/api/event";
import { useRef } from "react";

/**
 * 订阅 Tauri 事件，组件卸载时自动解绑。
 * listen 返回 Promise<UnlistenFn>，注册期间组件可能已卸载，因此用 cancelled 标记拦截「迟到的注册」，避免泄漏监听器。
 */
export function useTauriListen<T = unknown>(
  event: string,
  handler: (payload: T) => void,
): void {
  const unlistenRef = useRef<UnlistenFn | undefined>(void 0);

  useMount(async () => {
    unlistenRef.current = await listen<T>(event, (e) => {
      handler(e.payload);
    });
  });

  useUnmount(() => {
    unlistenRef.current?.();
  });
}
