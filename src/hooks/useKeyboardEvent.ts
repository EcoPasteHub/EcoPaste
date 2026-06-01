import { listen } from "@tauri-apps/api/event";
import { useEventListener } from "ahooks";
import { useEffect } from "react";
import { TAURI_EVENT } from "@/constants/events";
import { isWinMainWindow } from "@/utils/is";
import { log } from "@/utils/log";

type KeyboardEventType = "keydown" | "keyup";

interface NavEventPayload {
  action:
    | "up"
    | "down"
    | "enter"
    | "escape"
    | "nextTab"
    | "prevTab"
    | "ctrlDown"
    | "ctrlUp"
    | "shortcut";
  key?: string;
}

/**
 * 将 Rust 侧 nav action 映射为标准 KeyboardEvent 初始化参数。
 */
const navActionToKeyboardInit = (
  payload: NavEventPayload,
): (KeyboardEventInit & { type: KeyboardEventType }) | null => {
  switch (payload.action) {
    case "up":
      return { key: "ArrowUp", type: "keydown" };
    case "down":
      return { key: "ArrowDown", type: "keydown" };
    case "enter":
      return { key: "Enter", type: "keydown" };
    case "escape":
      return { key: "Escape", type: "keydown" };
    case "nextTab":
      return { key: "Tab", type: "keydown" };
    case "prevTab":
      return { key: "Tab", shiftKey: true, type: "keydown" };
    case "ctrlDown":
      return { ctrlKey: true, key: "Control", type: "keydown" };
    case "ctrlUp":
      return { ctrlKey: false, key: "Control", type: "keyup" };
    case "shortcut":
      if (!payload.key) return null;

      return { ctrlKey: true, key: payload.key, type: "keydown" };
    default:
      return null;
  }
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

  // macOS 或 Windows 非主窗口：直接用 window 事件
  useEventListener(type, handler, { enable: !needsRustNavEvent });

  // Windows 主窗口：监听 Tauri nav 事件，转换成 KeyboardEvent 格式
  useEffect(() => {
    if (!needsRustNavEvent) return;

    let unlistenFn: (() => void) | undefined;

    const setupListener = async () => {
      try {
        unlistenFn = await listen<NavEventPayload>(
          TAURI_EVENT.KEYBOARD_NAV,
          ({ payload }) => {
            const keyboardInit = navActionToKeyboardInit(payload);
            if (!keyboardInit) return;
            if (keyboardInit.type !== type) return;

            const syntheticEvent = new KeyboardEvent(keyboardInit.type, {
              bubbles: true,
              cancelable: true,
              ctrlKey: keyboardInit.ctrlKey,
              key: keyboardInit.key,
              shiftKey: keyboardInit.shiftKey,
            });

            handler(syntheticEvent);
          },
        );
      } catch (err) {
        log.error("Failed to listen keyboard nav event", err);
      }
    };

    setupListener();

    return () => {
      if (unlistenFn) {
        unlistenFn();
      }
    };
  }, [type, handler, needsRustNavEvent]);
};
