import type { CSSProperties } from "react";
import type { ClipboardPreviewRect, ClipboardPreviewState } from "@/commands";
import {
  PREVIEW_PANEL_GAP,
  PREVIEW_PANEL_MARGIN,
  PREVIEW_PANEL_MAX_HEIGHT,
  PREVIEW_PANEL_MAX_WIDTH,
  PREVIEW_PANEL_MIN_HEIGHT,
  PREVIEW_PANEL_MIN_WIDTH,
} from "./constants";
import type { PreviewMeasuredSize } from "./measurement";

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
