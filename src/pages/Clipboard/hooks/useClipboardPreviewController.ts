import { useEventListener } from "ahooks";
import type { PointerEvent as ReactPointerEvent } from "react";
import { useEffect, useRef, useState } from "react";
import { useSnapshot } from "valtio";
import { showClipboardPreview } from "@/commands";
import { TAURI_EVENT } from "@/constants/events";
import { useKeyboardEvent } from "@/hooks/useKeyboardEvent";
import { useTauriListen } from "@/hooks/useTauriListen";
import { settingsState } from "@/stores/settings";
import type { ClipboardItem } from "@/types/clipboard";
import { log } from "@/utils/log";
import {
  clearHoverTimer,
  closeClipboardPreviewSilently,
  HOVER_DELAY_MS,
  HOVER_HIDE_BUFFER_MS,
  isSpaceKey,
  type PreviewSession,
  type PreviewTrigger,
  type UseClipboardPreviewControllerOptions,
  type WindowVisibilityPayload,
} from "./previewController";

/**
 * 管理剪贴板预览的 hover / keyboard 生命周期和窗口隐藏兜底。
 */
export function useClipboardPreviewController(
  options: UseClipboardPreviewControllerOptions,
) {
  const { getActiveItem, itemElementMapRef, onHoverSelect } = options;
  const [previewSession, setPreviewSession] = useState<PreviewSession | null>(
    null,
  );
  const hoverTimerRef = useRef<number | null>(null);
  const hoverHideTimerRef = useRef<number | null>(null);
  const mainWindowVisibleRef = useRef(true);
  const previewSessionRef = useRef<PreviewSession | null>(null);
  const previewOpenRequestIdRef = useRef(0);
  const previewMoveFrameRef = useRef<number | null>(null);
  const previewMoveTargetRef = useRef<{
    item: ClipboardItem;
    pointerY: number;
  } | null>(null);
  const pendingHoverTargetRef = useRef<{
    item: ClipboardItem;
    pointerY: number;
  } | null>(null);
  const settingsSnapshot = useSnapshot(settingsState);
  const previewSettings = settingsSnapshot.clipboard.preview;

  useEffect(() => {
    if (!previewSettings.hoverEnabled && previewSession?.trigger === "hover") {
      clearHoverTimer(hoverTimerRef);
      clearHoverTimer(hoverHideTimerRef);
      previewSessionRef.current = null;
      setPreviewSession(null);
      closeClipboardPreviewSilently("hoverDisabled");
      return;
    }

    if (
      !previewSettings.spaceEnabled &&
      previewSession?.trigger === "keyboard"
    ) {
      clearHoverTimer(hoverTimerRef);
      clearHoverTimer(hoverHideTimerRef);
      previewSessionRef.current = null;
      setPreviewSession(null);
      closeClipboardPreviewSilently("spaceDisabled");
    }
  }, [
    previewSettings.hoverEnabled,
    previewSettings.spaceEnabled,
    previewSession?.trigger,
  ]);

  useEffect(() => {
    return () => {
      clearHoverTimer(hoverTimerRef);
      clearHoverTimer(hoverHideTimerRef);
      pendingHoverTargetRef.current = null;
      if (previewMoveFrameRef.current !== null) {
        window.cancelAnimationFrame(previewMoveFrameRef.current);
        previewMoveFrameRef.current = null;
        previewMoveTargetRef.current = null;
      }
    };
  }, []);

  const handleWindowVisibility = (event: {
    payload: WindowVisibilityPayload;
  }) => {
    if (event.payload.label !== "main") return;

    mainWindowVisibleRef.current = event.payload.visible;
    if (event.payload.visible) return;

    closePreview("windowHidden");
  };

  useTauriListen(TAURI_EVENT.WINDOW_VISIBILITY, handleWindowVisibility);

  const handleWindowBlur = () => {
    if (!previewSessionRef.current) return;

    closePreview("windowBlur");
  };

  useEventListener("blur", handleWindowBlur, { target: window });

  const handleWindowResize = () => {
    closePreview("windowResize");
  };

  useEventListener("resize", handleWindowResize, { target: window });

  /**
   * 关闭预览窗口并清理本地预览会话。
   */
  const closePreview = (reason: string) => {
    previewOpenRequestIdRef.current += 1;
    cancelHoverPreview();
    cancelHoverHide();
    cancelPreviewMoveFrame();
    commitPreviewSession(null);
    closeClipboardPreviewSilently(reason);
  };

  /**
   * Hover 进入：初次打开走 delay；已有 hover 预览时直接重定向到新条目。
   */
  const handleItemPointerEnter = (
    item: ClipboardItem,
    event: ReactPointerEvent<HTMLDivElement>,
  ) => {
    const pointerY = event.clientY;

    onHoverSelect(item.id);

    if (!mainWindowVisibleRef.current) return;
    if (!previewSettings.hoverEnabled) return;
    if (previewSessionRef.current?.trigger === "keyboard") return;

    cancelHoverPreview();
    cancelHoverHide();
    pendingHoverTargetRef.current = { item, pointerY };

    if (previewSessionRef.current?.trigger === "hover") {
      if (previewSessionRef.current.itemId === item.id) return;

      void openPreviewForItem(item, "hover", pointerY);
      return;
    }

    hoverTimerRef.current = window.setTimeout(() => {
      hoverTimerRef.current = null;

      if (!mainWindowVisibleRef.current) return;
      if (!settingsState.clipboard.preview.hoverEnabled) return;

      const target = pendingHoverTargetRef.current;

      if (!target || target.item.id !== item.id) return;

      void openPreviewForItem(target.item, "hover", target.pointerY);
    }, HOVER_DELAY_MS[previewSettings.hoverDelayMs]);
  };

  /**
   * Hover 预览打开后，同一卡片内移动鼠标会持续重定向锚点和预览卡片。
   */
  const handleItemPointerMove = (
    item: ClipboardItem,
    event: ReactPointerEvent<HTMLDivElement>,
  ) => {
    const pointerY = event.clientY;

    if (hoverTimerRef.current !== null) {
      pendingHoverTargetRef.current = { item, pointerY };
    }

    if (previewSessionRef.current?.trigger !== "hover") return;
    if (previewSessionRef.current.itemId !== item.id) return;

    previewMoveTargetRef.current = {
      item,
      pointerY,
    };

    if (previewMoveFrameRef.current !== null) return;

    previewMoveFrameRef.current = window.requestAnimationFrame(() => {
      previewMoveFrameRef.current = null;

      const target = previewMoveTargetRef.current;
      previewMoveTargetRef.current = null;

      if (!target) return;
      if (previewSessionRef.current?.trigger !== "hover") return;
      if (previewSessionRef.current.itemId !== target.item.id) return;

      void openPreviewForItem(target.item, "hover", target.pointerY);
    });
  };

  /**
   * Hover 离开单个卡片时进入准备隐藏状态；进入新卡片会取消隐藏。
   */
  const handleItemPointerLeave = () => {
    scheduleHoverHide("itemPointerLeave");
  };

  /**
   * 指针离开列表区域时结束 hover preview，卡片间移动只做 retarget。
   */
  const handlePreviewAreaPointerLeave = () => {
    scheduleHoverHide("hoverAreaLeave");
  };

  /**
   * 指针离开当前 document/window 时兜底关闭，避免 webview 边界漏掉元素级 leave。
   */
  const handleDocumentPointerOut = (event: PointerEvent) => {
    if (event.relatedTarget !== null) return;

    scheduleHoverHide("documentPointerOut");
  };

  useEventListener("pointerout", handleDocumentPointerOut, {
    target: document,
  });

  /**
   * 浏览器取消后续指针事件时关闭 hover preview，避免预览残留。
   */
  const handleDocumentPointerCancel = () => {
    scheduleHoverHide("documentPointerCancel");
  };

  useEventListener("pointercancel", handleDocumentPointerCancel, {
    target: document,
  });

  /**
   * Space 按下打开当前 active item；忽略重复 keydown，避免重复 IPC。
   */
  const handlePreviewSpaceDown = (event: KeyboardEvent) => {
    if (!previewSettings.spaceEnabled) return;
    if (event.repeat && previewSession?.trigger === "keyboard") return;

    event.preventDefault();
    cancelHoverPreview();
    cancelHoverHide();

    const activeItem = getActiveItem();

    if (!activeItem) return;

    void openPreviewForItem(activeItem, "keyboard");
  };

  /**
   * Space 松开关闭 keyboard preview。
   */
  const handlePreviewSpaceUp = (event: KeyboardEvent) => {
    if (!isSpaceKey(event)) return;

    event.preventDefault();

    if (previewSession?.trigger !== "keyboard") return;

    closePreview("spaceUp");
  };

  useKeyboardEvent("keyup", handlePreviewSpaceUp);

  /**
   * 方向键移动到新 active item 时同步 keyboard preview。
   */
  const handleKeyboardPreviewMove = (item: ClipboardItem) => {
    if (previewSession?.trigger !== "keyboard") return;

    void openPreviewForItem(item, "keyboard");
  };

  /**
   * 取消等待中的 hover preview。
   */
  const cancelHoverPreview = () => {
    clearHoverTimer(hoverTimerRef);
    pendingHoverTargetRef.current = null;
  };

  /**
   * 滚动列表时关闭 hover preview，保留 keyboard preview。
   */
  const closeHoverPreviewForScroll = () => {
    if (previewSession?.trigger === "hover") {
      closePreview("scroll");
      return;
    }

    cancelHoverPreview();
  };

  /**
   * 打开或重定向指定条目的预览 overlay。
   */
  const openPreviewForItem = async (
    item: ClipboardItem,
    trigger: PreviewTrigger,
    pointerY?: number,
  ) => {
    if (!mainWindowVisibleRef.current) return;

    const element = itemElementMapRef.current.get(item.id);
    const rect = element?.getBoundingClientRect();

    if (!rect || rect.width <= 0 || rect.height <= 0) return;

    const requestId = previewOpenRequestIdRef.current + 1;
    previewOpenRequestIdRef.current = requestId;
    commitPreviewSession({ itemId: item.id, trigger });

    try {
      if (!mainWindowVisibleRef.current) return;

      const state = await showClipboardPreview(item.id, {
        height: rect.height,
        left: rect.left,
        pointerY,
        top: rect.top,
        width: rect.width,
      });

      if (requestId !== previewOpenRequestIdRef.current) return;
      if (!state || !mainWindowVisibleRef.current) {
        closePreview("previewShowSuppressed");
      }
    } catch (error) {
      if (requestId === previewOpenRequestIdRef.current) {
        log.error("show clipboard preview failed", { error, trigger });
        commitPreviewSession(null);
      }
      return;
    }
  };

  /**
   * 取消等待中的 hover 隐藏缓冲。
   */
  const cancelHoverHide = () => {
    clearHoverTimer(hoverHideTimerRef);
  };

  /**
   * 结束 hover preview；keyboard preview 不受指针离开影响。
   */
  const closeHoverPreview = (reason: string) => {
    cancelHoverPreview();
    cancelHoverHide();

    if (previewSessionRef.current?.trigger !== "hover") return;

    closePreview(reason);
  };

  /**
   * 鼠标离开剪贴板项后进入准备隐藏状态，短时间内进入新项会取消隐藏。
   */
  const scheduleHoverHide = (reason: string) => {
    cancelHoverPreview();
    cancelHoverHide();

    if (previewSessionRef.current?.trigger !== "hover") return;

    hoverHideTimerRef.current = window.setTimeout(() => {
      hoverHideTimerRef.current = null;
      closeHoverPreview(reason);
    }, HOVER_HIDE_BUFFER_MS);
  };

  /**
   * 取消等待中的 mousemove retarget 帧。
   */
  function cancelPreviewMoveFrame() {
    if (previewMoveFrameRef.current === null) return;

    window.cancelAnimationFrame(previewMoveFrameRef.current);
    previewMoveFrameRef.current = null;
    previewMoveTargetRef.current = null;
    pendingHoverTargetRef.current = null;
  }

  /**
   * 同步预览会话 state 与 ref，供 hover 定时器和快速切换时读取最新会话。
   */
  function commitPreviewSession(session: PreviewSession | null) {
    previewSessionRef.current = session;
    setPreviewSession(session);
  }

  return {
    closeHoverPreviewForScroll,
    closePreview,
    handleItemPointerEnter,
    handleItemPointerLeave,
    handleItemPointerMove,
    handleKeyboardPreviewMove,
    handlePreviewAreaPointerLeave,
    handlePreviewSpaceDown,
    previewSession,
  };
}

/**
 * 判断 Space 键。
 */
export { isSpaceKey } from "./previewController";
