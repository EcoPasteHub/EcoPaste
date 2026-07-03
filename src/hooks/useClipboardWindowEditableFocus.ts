import { useEffect } from "react";
import { setClipboardWindowEditing } from "@/commands";
import { isWinClipboardWindow } from "@/utils/is";

const EDITABLE_BLUR_RESTORE_DELAY_MS = 80;

export const prepareClipboardWindowEditableFocus = async () => {
  if (!isWinClipboardWindow()) return;

  await setClipboardWindowEditing(true);
};

/**
 * Windows 剪贴板窗口输入控件激活期间临时允许窗口聚焦，编辑结束后恢复不可聚焦。
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
        if (findEditableElement(document.activeElement)) return;

        void setEditing(false);
      }, EDITABLE_BLUR_RESTORE_DELAY_MS);
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
    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      clearRestoreTimer();
      window.removeEventListener("pointerdown", handlePointerDown, true);
      window.removeEventListener("focusin", handleFocusIn, true);
      window.removeEventListener("focusout", scheduleRestore, true);
      window.removeEventListener("blur", scheduleRestore);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      void setClipboardWindowEditing(false);
    };
  }, []);
};

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
