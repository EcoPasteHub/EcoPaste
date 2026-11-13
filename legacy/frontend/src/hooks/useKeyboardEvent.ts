import { useEventListener, useLatest } from "ahooks";
import { TAURI_EVENT } from "@/constants/events";
import { isWinClipboardWindow } from "@/utils/is";
import { useTauriListen } from "./useTauriListen";

type KeyboardEventType = "keydown" | "keyup";

const EDITABLE_GLOBAL_KEYBOARD_ATTRIBUTE = "data-allow-global-keyboard";
const EDITABLE_GLOBAL_KEYBOARD_SELECTOR = `[${EDITABLE_GLOBAL_KEYBOARD_ATTRIBUTE}="true"]`;
const EDITABLE_GLOBAL_HANDOFF_KEYS = new Set([
  "ArrowDown",
  "ArrowLeft",
  "ArrowRight",
  "ArrowUp",
  "Control",
  "Enter",
  "Escape",
  "Tab",
]);

interface NavEventPayload {
  code?: string;
  type: KeyboardEventType;
  key: string;
  ctrlKey?: boolean;
  shiftKey?: boolean;
}

/**
 * 跨平台键盘事件监听 hook。
 *
 * macOS 与可聚焦窗口直接监听浏览器键盘事件；Windows 剪贴板窗口默认不可聚焦，
 * 导航键通常来自 Rust 低级钩子，但输入控件仍保留浏览器原生输入行为。
 */
export const useKeyboardEvent = (
  type: KeyboardEventType,
  handler: (event: KeyboardEvent) => void,
) => {
  const isWindowsClipboardWindow = isWinClipboardWindow();
  const handlerRef = useLatest(handler);

  const handleBrowserEvent = (event: KeyboardEvent) => {
    if (isWindowsClipboardWindow) {
      const editableTarget = findEditableElement(event.target);
      if (editableTarget) {
        if (!shouldHandoffEditableKeyboard(editableTarget, event)) return;

        editableTarget.blur();
      }
    }

    handlerRef.current(event);
  };

  useEventListener(type, handleBrowserEvent);

  useTauriListen<NavEventPayload>(TAURI_EVENT.KEYBOARD_NAV, (event) => {
    if (!isWindowsClipboardWindow) return;
    if (shouldUseNativeEditableKeyboard(document.activeElement)) return;

    const { type: payloadType, ...rest } = event.payload;

    if (payloadType !== type || !rest.key) return;

    handlerRef.current(
      new KeyboardEvent(payloadType, { cancelable: true, ...rest }),
    );
  });
};

function findEditableElement(target: EventTarget | null): HTMLElement | null {
  if (!(target instanceof Element)) return null;

  let element: Element | null = target;
  while (element) {
    if (element instanceof HTMLElement) {
      if (element.isContentEditable) return element;

      const tagName = element.tagName.toLowerCase();
      if (tagName === "input" || tagName === "textarea") return element;
    }

    element = element.parentElement;
  }

  return null;
}

function shouldUseNativeEditableKeyboard(target: EventTarget | null) {
  return findEditableElement(target) !== null;
}

function shouldHandoffEditableKeyboard(
  target: HTMLElement,
  event: KeyboardEvent,
) {
  if (event.type !== "keydown") return false;
  if (!target.closest(EDITABLE_GLOBAL_KEYBOARD_SELECTOR)) return false;

  return EDITABLE_GLOBAL_HANDOFF_KEYS.has(event.key);
}
