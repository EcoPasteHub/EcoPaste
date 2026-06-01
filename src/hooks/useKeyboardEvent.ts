import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { useEventListener } from "ahooks";
import { useEffect } from "react";
import { TAURI_EVENT } from "@/constants/events";
import { isWinMainWindow } from "@/utils/is";
import { log } from "@/utils/log";

type KeyboardEventType = "keydown" | "keyup";

interface NavEventPayload {
  action: "up" | "down" | "enter" | "escape" | "nextTab" | "prevTab";
}

/**
 * 将 Rust 侧 nav action 映射为标准 KeyboardEvent.key 值。
 */
const navActionToKey = (action: string): string => {
  switch (action) {
    case "up":
      return "ArrowUp";
    case "down":
      return "ArrowDown";
    case "enter":
      return "Enter";
    case "escape":
      return "Escape";
    case "nextTab":
      return "Tab";
    case "prevTab":
      return "Tab";
    default:
      return action;
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
    if (!needsRustNavEvent || type !== "keydown") return;

    let unlistenFn: (() => void) | undefined;

    const setupListener = async () => {
      try {
        unlistenFn = await getCurrentWebviewWindow().listen<NavEventPayload>(
          TAURI_EVENT.KEYBOARD_NAV,
          ({ payload }) => {
            const syntheticEvent = new KeyboardEvent("keydown", {
              bubbles: true,
              cancelable: true,
              key: navActionToKey(payload.action),
              shiftKey: payload.action === "prevTab",
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
