import { useEffect } from "react";
import { setClipboardWindowEditing } from "@/commands";
import { isWinClipboardWindow } from "@/utils/is";

const EDITABLE_BLUR_RESTORE_DELAY_MS = 80;

export const prepareClipboardWindowEditableFocus = async () => {
  if (!isWinClipboardWindow()) return;

  await setClipboardWindowEditing(true);
};

/**
 * Windows 剪贴板窗口在编辑或选中文本期间临时允许聚焦，保留 WebView 的原生复制。
 */
export const useClipboardWindowEditableFocus = () => {
  useEffect(() => {
    if (!isWinClipboardWindow()) return;

    let editing = false;
    let restoreTimer = 0;

    const clearRestoreTimer = () => {
      if (restoreTimer === 0) return;

      window.clearTimeout(restoreTimer);
      restoreTimer = 0;
    };

    const setEditing = async (nextEditing: boolean) => {
      if (editing === nextEditing) return;

      editing = nextEditing;
      await setClipboardWindowEditing(nextEditing);
    };

    const activateEditableTarget = async (target: HTMLElement) => {
      clearRestoreTimer();
      await setEditing(true);

      if (!document.contains(target)) return;
      if (document.activeElement === target) return;
      if (findEditableElement(document.activeElement)) return;

      target.focus();
    };

    const handlePointerDown = (event: PointerEvent) => {
      const target = findEditableElement(event.target);
      if (!target) return;

      void activateEditableTarget(target);
    };

    const handleFocusIn = (event: FocusEvent) => {
      const target = findEditableElement(event.target);
      if (!target) return;

      clearRestoreTimer();
      void setEditing(true);
    };

    const scheduleRestore = () => {
      clearRestoreTimer();

      restoreTimer = window.setTimeout(() => {
        restoreTimer = 0;
        if (findEditableElement(document.activeElement) || hasCardSelection()) {
          return;
        }

        void setEditing(false);
      }, EDITABLE_BLUR_RESTORE_DELAY_MS);
    };

    const handleSelectionChange = () => {
      if (hasCardSelection() && document.visibilityState === "visible") {
        clearRestoreTimer();
        void setEditing(true);
        return;
      }

      if (editing) scheduleRestore();
    };

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") return;

      clearRestoreTimer();
      void setEditing(false);
    };

    window.addEventListener("pointerdown", handlePointerDown, true);
    window.addEventListener("focusin", handleFocusIn, true);
    window.addEventListener("focusout", scheduleRestore, true);
    window.addEventListener("blur", scheduleRestore);
    document.addEventListener("selectionchange", handleSelectionChange);
    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      clearRestoreTimer();
      window.removeEventListener("pointerdown", handlePointerDown, true);
      window.removeEventListener("focusin", handleFocusIn, true);
      window.removeEventListener("focusout", scheduleRestore, true);
      window.removeEventListener("blur", scheduleRestore);
      document.removeEventListener("selectionchange", handleSelectionChange);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      void setClipboardWindowEditing(false);
    };
  }, []);
};

/**
 * 只有起止点在同一卡片的正文内，才为原生 Ctrl+C 保持窗口聚焦。
 */
function hasCardSelection() {
  const selection = window.getSelection();
  if (!selection || selection.isCollapsed || selection.rangeCount !== 1) {
    return false;
  }

  const { startContainer, endContainer } = selection.getRangeAt(0);
  const startElement =
    startContainer instanceof Element
      ? startContainer
      : startContainer.parentElement;
  const textRoot = startElement?.closest("[data-clipboard-text]");

  return textRoot?.contains(endContainer) ?? false;
}

function findEditableElement(target: EventTarget | null): HTMLElement | null {
  if (!(target instanceof Element)) return null;

  let element: Element | null = target;
  while (element) {
    if (element instanceof HTMLElement && isEditableElement(element)) {
      return element;
    }

    element = element.parentElement;
  }

  return null;
}

function isEditableElement(element: HTMLElement) {
  if (element.isContentEditable) return true;

  const tagName = element.tagName.toLowerCase();

  return tagName === "input" || tagName === "textarea";
}
