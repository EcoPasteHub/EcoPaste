import { listen, type UnlistenFn } from "@tauri-apps/api/event";
import { useEffect, useRef } from "react";

// @tauri-apps/api 的 listen 返回 Promise<UnlistenFn>，注册期间组件可能已卸载，
// 因此用 cancelled 标记拦截「迟到的注册」，避免泄漏监听器。
// handler 走 ref 转发，effect 仅依赖事件名。
export function useTauriListen<T = unknown>(
  event: string,
  handler: (payload: T) => void,
): void {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;

  useEffect(() => {
    let cancelled = false;
    let unlisten: UnlistenFn | undefined;

    listen<T>(event, (e) => {
      handlerRef.current(e.payload);
    }).then((fn) => {
      if (cancelled) {
        fn();
      } else {
        unlisten = fn;
      }
    });

    return () => {
      cancelled = true;
      unlisten?.();
    };
  }, [event]);
}
