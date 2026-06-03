import type { RefObject } from "react";
import { closeClipboardPreview } from "@/commands";
import type { ClipboardItem } from "@/types/clipboard";
import type { PreviewHoverDelayMs } from "@/types/settings";
import { log } from "@/utils/log";

export const HOVER_DELAY_MS: Record<PreviewHoverDelayMs, number> = {
  ms300: 300,
  ms500: 500,
  ms1000: 1000,
};
export const HOVER_HIDE_BUFFER_MS = 240;

export type PreviewTrigger = "keyboard" | "hover";

export interface PreviewSession {
  itemId: string;
  trigger: PreviewTrigger;
}

export interface WindowVisibilityPayload {
  label: string;
  visible: boolean;
}

export interface UseClipboardPreviewControllerOptions {
  getActiveItem: () => ClipboardItem | null;
  itemElementMapRef: RefObject<Map<string, HTMLDivElement>>;
  onHoverSelect: (id: string) => void;
}

/**
 * 判断 Space 键，兼容旧版 WebKit 的 Spacebar 命名。
 */
export function isSpaceKey(event: KeyboardEvent) {
  return event.key === " " || event.key === "Spacebar";
}

/**
 * 清理 hover 延迟任务。
 */
export function clearHoverTimer(timerRef: { current: number | null }) {
  if (timerRef.current === null) return;

  window.clearTimeout(timerRef.current);
  timerRef.current = null;
}

/**
 * 非关键路径关闭预览，失败只写日志。
 */
export async function closeClipboardPreviewSilently(reason: string) {
  try {
    await closeClipboardPreview();
  } catch (error) {
    log.error("close clipboard preview failed", { error, reason });
  }
}
