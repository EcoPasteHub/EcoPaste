import { useEffect, useRef } from "react";
import { listen } from "tauri-awesome-rpc";

// awesome-rpc 的 listen 是同步注册（WS 通道直挂 window.AwesomeListener），
// 返回值就是 UnlistenFn —— 不像官方 @tauri-apps/api/event 那样是 Promise，不必处理「注册期间组件已卸载」的竞态。
// payload 走 JSON-RPC 是 unknown，由调用方用泛型断言；handler 走 ref 转发，effect 仅依赖事件名。
export function useTauriListen<T = unknown>(
  event: string,
  handler: (payload: T) => void,
): void {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;

  useEffect(() => {
    const unlisten = listen(event, (payload) => {
      handlerRef.current(payload as T);
    });
    return unlisten;
  }, [event]);
}
