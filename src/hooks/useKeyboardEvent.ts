import { useEventListener, useLatest } from "ahooks";
import { TAURI_EVENT } from "@/constants/events";
import { isWinMainWindow } from "@/utils/is";
import { useTauriListen } from "./useTauriListen";

type KeyboardEventType = "keydown" | "keyup";

interface NavEventPayload {
  type: KeyboardEventType;
  key: string;
  ctrlKey?: boolean;
  shiftKey?: boolean;
}

/**
 * 跨平台键盘事件监听 hook。
 *
 * - macOS：直接监听 window 原生键盘事件（NSPanel 有焦点）
 * - Windows 主窗口：监听 Rust 通过低级钩子 emit 的 `keyboard://nav` 事件（focusable=false）
 * - Windows 其他窗口：直接监听 window 原生键盘事件（focusable=true）
 *
 * 对外暴露统一的 KeyboardEvent 接口，调用方无需关心平台差异。
 */
export const useKeyboardEvent = (
  type: KeyboardEventType,
  handler: (event: KeyboardEvent) => void,
) => {
  const needsRustNavEvent = isWinMainWindow();

  // useTauriListen 挂载时仅捕获一次回调，用 ref 确保始终调用最新 handler。
  const handlerRef = useLatest(handler);

  useEventListener(type, handler);

  useTauriListen<NavEventPayload>(TAURI_EVENT.KEYBOARD_NAV, (event) => {
    if (!needsRustNavEvent) return;

    const { type: payloadType, ...rest } = event.payload;

    if (payloadType !== type || !rest.key) return;

    handlerRef.current(
      new KeyboardEvent(payloadType, { cancelable: true, ...rest }),
    );
  });
};
