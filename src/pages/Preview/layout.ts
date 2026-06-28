import type { CSSProperties } from "react";
import type {
  ClipboardPreviewPayload,
  ClipboardPreviewRect,
  ClipboardPreviewState,
} from "@/commands";
import {
  PREVIEW_EMPTY_CONTENT_HEIGHT,
  PREVIEW_FILE_MORE_FOOTER_HEIGHT,
  PREVIEW_FILE_ROW_HEIGHT,
  PREVIEW_FILE_VERTICAL_PADDING,
  PREVIEW_PANEL_GAP,
  PREVIEW_PANEL_HEADER_HEIGHT,
  PREVIEW_PANEL_IMAGE_PADDING_X,
  PREVIEW_PANEL_IMAGE_PADDING_Y,
  PREVIEW_PANEL_MARGIN,
  PREVIEW_PANEL_MAX_HEIGHT,
  PREVIEW_PANEL_MAX_WIDTH,
  PREVIEW_PANEL_MIN_HEIGHT,
  PREVIEW_PANEL_MIN_WIDTH,
  PREVIEW_TEXT_ROW_HEIGHT,
  PREVIEW_TEXT_SOFT_WRAP_CHARS,
  PREVIEW_TEXT_VERTICAL_PADDING,
} from "./constants";
import type { PreviewMeasuredSize } from "./measurement";

/**
 * 图片 payload 已有 DB 尺寸时，直接按比例估算面板尺寸，避免等图片加载后才撑开。
 */
export function resolveEffectivePanelSize(
  layout: ClipboardPreviewState["layout"],
  measuredSize: PreviewMeasuredSize,
  payload: ClipboardPreviewPayload | null,
): PreviewMeasuredSize {
  if (!payload) return measuredSize;

  if (payload.kind === "text") {
    return resolveTextPanelSize(layout, payload.text ?? "");
  }

  if (payload.kind === "files") {
    return resolveFilesPanelSize(
      layout,
      payload.files.length,
      payload.totalFiles,
    );
  }

  if (payload.kind !== "image") return measuredSize;

  const imageSize = resolveImagePanelSize(
    layout,
    payload.imageWidth,
    payload.imageHeight,
  );
  if (!imageSize) return measuredSize;

  return imageSize;
}

/**
 * 根据内容自然尺寸生成实际面板 rect，保留 Rust 给出的 placement 和可用最大区域。
 */
export function resolveDynamicPanelRect(
  layout: ClipboardPreviewState["layout"],
  measuredSize: PreviewMeasuredSize,
) {
  const width = clamp(
    measuredSize.width,
    PREVIEW_PANEL_MIN_WIDTH,
    Math.min(PREVIEW_PANEL_MAX_WIDTH, layout.panelRect.width),
  );
  const height = clamp(
    measuredSize.height,
    PREVIEW_PANEL_MIN_HEIGHT,
    Math.min(PREVIEW_PANEL_MAX_HEIGHT, layout.panelRect.height),
  );
  const raw = rawDynamicPanelRect(layout, width, height);

  return clampRect(raw, insetRect(layout.overlayRect, PREVIEW_PANEL_MARGIN));
}

/**
 * 让隐藏测量层使用和真实面板一致的宽度上限，避免文本换行数被低估。
 */
export function resolveMeasurePanelStyle(
  layout: ClipboardPreviewState["layout"],
): CSSProperties {
  const maxWidth = Math.min(PREVIEW_PANEL_MAX_WIDTH, layout.panelRect.width);
  const minWidth = Math.min(PREVIEW_PANEL_MIN_WIDTH, maxWidth);

  return {
    maxWidth,
    minWidth,
  };
}

/**
 * 把跨端 rect 转成 React absolute positioning style。
 */
export function rectStyle(rect: ClipboardPreviewRect) {
  return {
    height: rect.height,
    left: rect.left,
    top: rect.top,
    width: rect.width,
  };
}

/**
 * 按 placement 把动态面板仍放在列表项对应侧，并围绕 source 中线对齐。
 */
function rawDynamicPanelRect(
  layout: ClipboardPreviewState["layout"],
  width: number,
  height: number,
) {
  const sourceRect = layout.sourceRect;
  const centeredTop = sourceRect.top + sourceRect.height / 2 - height / 2;
  const centeredLeft = sourceRect.left + sourceRect.width / 2 - width / 2;

  switch (layout.placement) {
    case "right":
      return {
        height,
        left: sourceRect.left + sourceRect.width + PREVIEW_PANEL_GAP,
        top: centeredTop,
        width,
      };
    case "left":
      return {
        height,
        left: sourceRect.left - PREVIEW_PANEL_GAP - width,
        top: centeredTop,
        width,
      };
    case "bottom":
      return {
        height,
        left: centeredLeft,
        top: sourceRect.top + sourceRect.height + PREVIEW_PANEL_GAP,
        width,
      };
    case "top":
      return {
        height,
        left: centeredLeft,
        top: sourceRect.top - PREVIEW_PANEL_GAP - height,
        width,
      };
  }
}

/**
 * 按图片原始宽高、面板上限和固定内边距计算等比例展示尺寸。
 */
function resolveImagePanelSize(
  layout: ClipboardPreviewState["layout"],
  imageWidth: number | null,
  imageHeight: number | null,
): PreviewMeasuredSize | null {
  if (!imageWidth || !imageHeight || imageWidth <= 0 || imageHeight <= 0) {
    return null;
  }

  const maxPanelWidth = Math.min(
    PREVIEW_PANEL_MAX_WIDTH,
    layout.panelRect.width,
  );
  const maxPanelHeight = Math.min(
    PREVIEW_PANEL_MAX_HEIGHT,
    layout.panelRect.height,
  );
  const maxImageWidth = Math.max(
    1,
    maxPanelWidth - PREVIEW_PANEL_IMAGE_PADDING_X,
  );
  const maxImageHeight = Math.max(
    1,
    maxPanelHeight -
      PREVIEW_PANEL_HEADER_HEIGHT -
      PREVIEW_PANEL_IMAGE_PADDING_Y,
  );
  const scale = Math.min(
    1,
    maxImageWidth / imageWidth,
    maxImageHeight / imageHeight,
  );

  return {
    height: clamp(
      Math.ceil(imageHeight * scale) +
        PREVIEW_PANEL_HEADER_HEIGHT +
        PREVIEW_PANEL_IMAGE_PADDING_Y,
      PREVIEW_PANEL_MIN_HEIGHT,
      maxPanelHeight,
    ),
    width: clamp(
      Math.ceil(imageWidth * scale) + PREVIEW_PANEL_IMAGE_PADDING_X,
      PREVIEW_PANEL_MIN_WIDTH,
      maxPanelWidth,
    ),
  };
}

/**
 * 文本 viewer 使用虚拟行渲染，测量层不再渲染整段内容；这里按同款软切行估算面板尺寸。
 */
function resolveTextPanelSize(
  layout: ClipboardPreviewState["layout"],
  text: string,
): PreviewMeasuredSize {
  const maxPanelWidth = Math.min(
    PREVIEW_PANEL_MAX_WIDTH,
    layout.panelRect.width,
  );
  const maxPanelHeight = Math.min(
    PREVIEW_PANEL_MAX_HEIGHT,
    layout.panelRect.height,
  );
  const rowCount = countTextPreviewRows(text);
  const contentHeight =
    rowCount === 0
      ? PREVIEW_EMPTY_CONTENT_HEIGHT
      : rowCount * PREVIEW_TEXT_ROW_HEIGHT + PREVIEW_TEXT_VERTICAL_PADDING;

  return {
    height: clamp(
      PREVIEW_PANEL_HEADER_HEIGHT + contentHeight,
      PREVIEW_PANEL_MIN_HEIGHT,
      maxPanelHeight,
    ),
    width: clamp(maxPanelWidth, PREVIEW_PANEL_MIN_WIDTH, maxPanelWidth),
  };
}

/**
 * 文件 viewer 也走虚拟列表；尺寸只按已返回的行数和截断提示估算。
 */
function resolveFilesPanelSize(
  layout: ClipboardPreviewState["layout"],
  shownCount: number,
  totalCount: number,
): PreviewMeasuredSize {
  const maxPanelWidth = Math.min(
    PREVIEW_PANEL_MAX_WIDTH,
    layout.panelRect.width,
  );
  const maxPanelHeight = Math.min(
    PREVIEW_PANEL_MAX_HEIGHT,
    layout.panelRect.height,
  );
  const contentHeight =
    shownCount === 0
      ? PREVIEW_EMPTY_CONTENT_HEIGHT
      : shownCount * PREVIEW_FILE_ROW_HEIGHT +
        PREVIEW_FILE_VERTICAL_PADDING +
        (totalCount > shownCount ? PREVIEW_FILE_MORE_FOOTER_HEIGHT : 0);

  return {
    height: clamp(
      PREVIEW_PANEL_HEADER_HEIGHT + contentHeight,
      PREVIEW_PANEL_MIN_HEIGHT,
      maxPanelHeight,
    ),
    width: clamp(maxPanelWidth, PREVIEW_PANEL_MIN_WIDTH, maxPanelWidth),
  };
}

/**
 * 统计虚拟文本行数，和 `TextViewer` 的软切块规则保持一致。
 */
function countTextPreviewRows(text: string) {
  if (text.length === 0) return 0;

  let rowCount = 0;
  for (const line of text.split("\n")) {
    rowCount += Math.max(
      1,
      Math.ceil(line.length / PREVIEW_TEXT_SOFT_WRAP_CHARS),
    );
  }

  return rowCount;
}

/**
 * 把 rect 限制在 bounds 内。
 */
function clampRect(rect: ClipboardPreviewRect, bounds: ClipboardPreviewRect) {
  const maxLeft = Math.max(bounds.left, rectRight(bounds) - rect.width);
  const maxTop = Math.max(bounds.top, rectBottom(bounds) - rect.height);

  return {
    height: rect.height,
    left: clamp(rect.left, bounds.left, maxLeft),
    top: clamp(rect.top, bounds.top, maxTop),
    width: rect.width,
  };
}

/**
 * 生成带安全边距的内部边界。
 */
function insetRect(rect: ClipboardPreviewRect, amount: number) {
  return {
    height: Math.max(1, rect.height - amount * 2),
    left: rect.left + amount,
    top: rect.top + amount,
    width: Math.max(1, rect.width - amount * 2),
  };
}

/**
 * 限制数值范围。
 */
function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

/**
 * 返回 rect 右边界。
 */
function rectRight(rect: { left: number; width: number }) {
  return rect.left + rect.width;
}

/**
 * 返回 rect 下边界。
 */
function rectBottom(rect: { top: number; height: number }) {
  return rect.top + rect.height;
}
