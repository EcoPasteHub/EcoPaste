import { useEventListener } from "ahooks";
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
 * 将 Rust nav payload 转成 KeyboardEvent 并分发到 window。
 */
const dispatchNavPayloadAsKeyboardEvent = (event: {
  payload: NavEventPayload;
}) => {
  const { type, ...rest } = event.payload;

  if (!rest.key) return;

  const syntheticEvent = new KeyboardEvent(type, {
    bubbles: true,
    cancelable: true,
    ...rest,
  });

  window.dispatchEvent(syntheticEvent);
};

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

  // 统一通过 window 事件分发，Windows 主窗口由 Rust 事件桥接补齐。
  useEventListener(type, handler);

  useTauriListen(
    TAURI_EVENT.KEYBOARD_NAV,
    needsRustNavEvent ? dispatchNavPayloadAsKeyboardEvent : () => void 0,
  );
};
