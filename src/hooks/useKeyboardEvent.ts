import { listen } from "@tauri-apps/api/event";
import { useEventListener, useMount, useUnmount } from "ahooks";
import { useRef } from "react";
import { TAURI_EVENT } from "@/constants/events";
import { isWinMainWindow } from "@/utils/is";
import { log } from "@/utils/log";

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
  const { ctrlKey, key, shiftKey, type } = event.payload;

  if (!key) return;

  const syntheticEvent = new KeyboardEvent(type, {
    bubbles: true,
    cancelable: true,
    ctrlKey,
    key,
    shiftKey,
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
  const unlistenRef = useRef<(() => void) | null>(null);

  // 统一通过 window 事件分发，Windows 主窗口由 Rust 事件桥接补齐。
  useEventListener(type, handler);

  /**
   * 注册 Windows 主窗口的 Rust 键盘事件监听。
   */
  const setupRustNavListener = async () => {
    if (!needsRustNavEvent) return;

    try {
      unlistenRef.current = await listen<NavEventPayload>(
        TAURI_EVENT.KEYBOARD_NAV,
        dispatchNavPayloadAsKeyboardEvent,
      );
    } catch (err) {
      log.error("Failed to listen keyboard nav event", err);
    }
  };

  /**
   * 注销 Rust 键盘事件监听，避免窗口销毁后残留订阅。
   */
  const cleanupRustNavListener = () => {
    const unlisten = unlistenRef.current;

    if (!unlisten) return;

    unlisten();

    unlistenRef.current = null;
  };

  useMount(setupRustNavListener);

  useUnmount(cleanupRustNavListener);
};
