import type { RefObject } from "react";
import { useEffect, useLayoutEffect, useState } from "react";
import { PREVIEW_PANEL_FALLBACK_SIZE } from "./constants";

export interface PreviewMeasuredSize {
  height: number;
  width: number;
}

/**
 * 监听隐藏测量层尺寸，并把内容自然尺寸同步给动态面板布局。
 */
export function useMeasuredPanelSize(
  ref: RefObject<HTMLDivElement | null>,
): PreviewMeasuredSize {
  const [size, setSize] = useState<PreviewMeasuredSize>(
    PREVIEW_PANEL_FALLBACK_SIZE,
  );

  useEffect(() => {
    const node = ref.current;
    if (!node) return;

    syncMeasuredPanelSize(node, setSize);

    const observer = new ResizeObserver(() => {
      syncMeasuredPanelSize(node, setSize);
    });
    observer.observe(node);

    return () => {
      observer.disconnect();
    };
  }, [ref]);

  useLayoutEffect(() => {
    const node = ref.current;
    if (!node) return;

    syncMeasuredPanelSize(node, setSize);
  });

  return size;
}

/**
 * 判断隐藏测量层是否已经给出当前内容的真实尺寸。
 */
export function hasMeasuredPanelSize(size: PreviewMeasuredSize) {
  return (
    size.height !== PREVIEW_PANEL_FALLBACK_SIZE.height ||
    size.width !== PREVIEW_PANEL_FALLBACK_SIZE.width
  );
}

/**
 * 从测量节点读取自然尺寸，相同尺寸不触发 state 更新。
 */
function syncMeasuredPanelSize(
  node: HTMLDivElement,
  setSize: React.Dispatch<React.SetStateAction<PreviewMeasuredSize>>,
) {
  const rect = node.getBoundingClientRect();
  const nextSize = {
    height: Math.ceil(rect.height),
    width: Math.ceil(rect.width),
  };

  setSize((currentSize) => {
    if (
      currentSize.height === nextSize.height &&
      currentSize.width === nextSize.width
    ) {
      return currentSize;
    }

    return nextSize;
  });
}
