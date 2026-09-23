import type { ClipboardUpdatedPayload } from "@/types/clipboard";

interface ImageOcrRefreshContext {
  atTop: boolean;
  enabled: boolean;
  keyword: string;
  visible: boolean;
}

/**
 * OCR changes alter search membership across every filter. Destructive index
 * changes must invalidate cached results immediately; worker progress waits
 * until refreshing will not interrupt scrolling or wake a hidden window.
 */
export function resolveImageOcrRefresh(
  change: NonNullable<ClipboardUpdatedPayload["ocr"]>,
  context: ImageOcrRefreshContext,
): "ignore" | "defer" | "reload" | "reset" {
  if (!context.keyword.trim()) return "ignore";
  if (change === "cleared") return "reset";
  if (!context.enabled) return "ignore";
  if (!context.visible || !context.atTop) return "defer";

  return "reload";
}
