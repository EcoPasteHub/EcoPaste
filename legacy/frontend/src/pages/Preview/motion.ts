import { useMotionValue, useSpring, useTransform } from "motion/react";
import { useLayoutEffect, useRef } from "react";
import { PREVIEW_SPRING } from "./constants";
import { buildConnectorPath, type PreviewConnector } from "./geometry";

interface PreviewPanelRect {
  height: number;
  left: number;
  top: number;
  width: number;
}

/**
 * 将最新 layout 写入 Motion Value；首次布局直接跳到目标，后续 retarget 走 spring。
 */
export function usePreviewMotion(
  active: boolean,
  sessionId: number | null,
  measured: boolean,
  panelRect: PreviewPanelRect,
  connector: PreviewConnector,
) {
  const activeSessionRef = useRef<number | null>(null);
  const jumpUntilMeasuredSessionRef = useRef<number | null>(null);
  const panelX = useMotionValue(panelRect.left);
  const panelY = useMotionValue(panelRect.top);
  const panelWidth = useMotionValue(panelRect.width);
  const panelHeight = useMotionValue(panelRect.height);
  const sourceXValue = useMotionValue(connector.source.x);
  const sourceYValue = useMotionValue(connector.source.y);
  const targetXValue = useMotionValue(connector.target.x);
  const targetYValue = useMotionValue(connector.target.y);
  const sourceDotXValue = useMotionValue(connector.sourceDot.x);
  const sourceDotYValue = useMotionValue(connector.sourceDot.y);
  const targetDotXValue = useMotionValue(connector.targetDot.x);
  const targetDotYValue = useMotionValue(connector.targetDot.y);
  const control1X = useMotionValue(connector.control1.x);
  const control1Y = useMotionValue(connector.control1.y);
  const control2X = useMotionValue(connector.control2.x);
  const control2Y = useMotionValue(connector.control2.y);
  const x = useSpring(panelX, PREVIEW_SPRING);
  const y = useSpring(panelY, PREVIEW_SPRING);
  const sourceX = useSpring(sourceXValue, PREVIEW_SPRING);
  const sourceY = useSpring(sourceYValue, PREVIEW_SPRING);
  const targetX = useSpring(targetXValue, PREVIEW_SPRING);
  const targetY = useSpring(targetYValue, PREVIEW_SPRING);
  const sourceDotX = useSpring(sourceDotXValue, PREVIEW_SPRING);
  const sourceDotY = useSpring(sourceDotYValue, PREVIEW_SPRING);
  const targetDotX = useSpring(targetDotXValue, PREVIEW_SPRING);
  const targetDotY = useSpring(targetDotYValue, PREVIEW_SPRING);
  const c1x = useSpring(control1X, PREVIEW_SPRING);
  const c1y = useSpring(control1Y, PREVIEW_SPRING);
  const c2x = useSpring(control2X, PREVIEW_SPRING);
  const c2y = useSpring(control2Y, PREVIEW_SPRING);
  const width = useSpring(panelWidth, PREVIEW_SPRING);
  const height = useSpring(panelHeight, PREVIEW_SPRING);

  useLayoutEffect(() => {
    if (!active) {
      activeSessionRef.current = null;
      jumpUntilMeasuredSessionRef.current = null;
      return;
    }
    if (sessionId === null) return;

    const isNewSession = activeSessionRef.current !== sessionId;
    const shouldJump =
      isNewSession || jumpUntilMeasuredSessionRef.current === sessionId;

    const values = [
      [panelX, panelRect.left],
      [panelY, panelRect.top],
      [panelWidth, panelRect.width],
      [panelHeight, panelRect.height],
      [sourceXValue, connector.source.x],
      [sourceYValue, connector.source.y],
      [targetXValue, connector.target.x],
      [targetYValue, connector.target.y],
      [sourceDotXValue, connector.sourceDot.x],
      [sourceDotYValue, connector.sourceDot.y],
      [targetDotXValue, connector.targetDot.x],
      [targetDotYValue, connector.targetDot.y],
      [control1X, connector.control1.x],
      [control1Y, connector.control1.y],
      [control2X, connector.control2.x],
      [control2Y, connector.control2.y],
    ] as const;

    for (const [value, next] of values) {
      if (shouldJump) {
        value.jump(next);
      } else {
        value.set(next);
      }
    }

    if (shouldJump) {
      const springValues = [
        [x, panelRect.left],
        [y, panelRect.top],
        [width, panelRect.width],
        [height, panelRect.height],
        [sourceX, connector.source.x],
        [sourceY, connector.source.y],
        [targetX, connector.target.x],
        [targetY, connector.target.y],
        [sourceDotX, connector.sourceDot.x],
        [sourceDotY, connector.sourceDot.y],
        [targetDotX, connector.targetDot.x],
        [targetDotY, connector.targetDot.y],
        [c1x, connector.control1.x],
        [c1y, connector.control1.y],
        [c2x, connector.control2.x],
        [c2y, connector.control2.y],
      ] as const;

      for (const [value, next] of springValues) {
        value.jump(next);
      }
    }

    if (isNewSession) {
      activeSessionRef.current = sessionId;
      jumpUntilMeasuredSessionRef.current = sessionId;
    } else if (measured) {
      jumpUntilMeasuredSessionRef.current = null;
    }
  }, [
    active,
    c1x,
    c1y,
    c2x,
    c2y,
    connector.control1.x,
    connector.control1.y,
    connector.control2.x,
    connector.control2.y,
    connector.source.x,
    connector.source.y,
    connector.sourceDot.x,
    connector.sourceDot.y,
    connector.target.x,
    connector.target.y,
    connector.targetDot.x,
    connector.targetDot.y,
    control1X,
    control1Y,
    control2X,
    control2Y,
    height,
    measured,
    panelHeight,
    panelRect.height,
    panelRect.left,
    panelRect.top,
    panelRect.width,
    panelWidth,
    panelX,
    panelY,
    sessionId,
    sourceDotX,
    sourceDotXValue,
    sourceDotY,
    sourceDotYValue,
    sourceX,
    sourceXValue,
    sourceY,
    sourceYValue,
    targetDotX,
    targetDotXValue,
    targetDotY,
    targetDotYValue,
    targetX,
    targetXValue,
    targetY,
    targetYValue,
    width,
    x,
    y,
  ]);

  const path = useTransform(
    [sourceX, sourceY, c1x, c1y, c2x, c2y, targetX, targetY],
    ([sx, sy, c1xValue, c1yValue, c2xValue, c2yValue, tx, ty]) => {
      return buildConnectorPath(
        { x: toMotionNumber(sx), y: toMotionNumber(sy) },
        { x: toMotionNumber(c1xValue), y: toMotionNumber(c1yValue) },
        { x: toMotionNumber(c2xValue), y: toMotionNumber(c2yValue) },
        { x: toMotionNumber(tx), y: toMotionNumber(ty) },
      );
    },
  );

  return {
    panelStyle: {
      height,
      width,
      x,
      y,
    },
    path,
    sourceDotX,
    sourceDotY,
    sourceX,
    sourceY,
    targetDotX,
    targetDotY,
    targetX,
    targetY,
  };
}

/**
 * Motion 的多输入 transform 回调类型为 unknown，这里收窄为数值坐标。
 */
function toMotionNumber(value: unknown) {
  return typeof value === "number" ? value : 0;
}
