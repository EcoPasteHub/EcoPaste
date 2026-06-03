import type { ClipboardPreviewRect } from "@/commands";

type ConnectorSide = "left" | "right" | "top" | "bottom";

interface Point {
  x: number;
  y: number;
}

interface AnchorCandidate {
  point: Point;
  side: ConnectorSide;
}

export interface PreviewConnector {
  source: Point;
  target: Point;
  sourceDot: Point;
  targetDot: Point;
  control1: Point;
  control2: Point;
  sourceSide: ConnectorSide;
  targetSide: ConnectorSide;
  path: string;
}

const CONTROL_RATIO = 0.22;
const MIN_CONTROL_DISTANCE = 20;
const MAX_CONTROL_DISTANCE = 64;

/**
 * 根据列表项矩形和预览面板矩形生成 cubic Bezier 连接线。
 */
export function resolveConnector(
  sourceRect: ClipboardPreviewRect,
  targetRect: ClipboardPreviewRect,
): PreviewConnector {
  const candidates = buildAnchorCandidates(sourceRect).flatMap((source) => {
    return buildAnchorCandidates(targetRect).map((target) => {
      return { score: scoreCandidate(source, target), source, target };
    });
  });
  const best = candidates.reduce((current, next) => {
    return next.score < current.score ? next : current;
  });
  const sourceDirection = normalForSide(best.source.side);
  const targetDirection = normalForSide(best.target.side);
  const source = best.source.point;
  const target = best.target.point;
  const distance = pointDistance(source, target);
  const controlDistance = clamp(
    distance * CONTROL_RATIO,
    MIN_CONTROL_DISTANCE,
    MAX_CONTROL_DISTANCE,
  );
  const control1 = project(source, sourceDirection, controlDistance);
  const control2 = project(target, targetDirection, controlDistance);
  const path = buildConnectorPath(source, control1, control2, target);

  return {
    control1,
    control2,
    path,
    source,
    sourceDot: source,
    sourceSide: best.source.side,
    target,
    targetDot: target,
    targetSide: best.target.side,
  };
}

/**
 * 用固定 `M C` 结构生成 SVG path，便于 Motion Value 持续更新同一条曲线。
 */
export function buildConnectorPath(
  source: Point,
  control1: Point,
  control2: Point,
  target: Point,
) {
  return [
    `M ${formatPoint(source)}`,
    `C ${formatPoint(control1)} ${formatPoint(control2)} ${formatPoint(target)}`,
  ].join(" ");
}

/**
 * 把 rect 转成四边中点候选锚点。
 */
function buildAnchorCandidates(rect: ClipboardPreviewRect): AnchorCandidate[] {
  const centerX = rect.left + rect.width / 2;
  const centerY = rect.top + rect.height / 2;
  const right = rect.left + rect.width;
  const bottom = rect.top + rect.height;

  return [
    { point: { x: rect.left, y: centerY }, side: "left" },
    { point: { x: right, y: centerY }, side: "right" },
    { point: { x: centerX, y: rect.top }, side: "top" },
    { point: { x: centerX, y: bottom }, side: "bottom" },
  ];
}

/**
 * 给一组 source/target 锚点打分，优先短距离和外法线方向一致的连接。
 */
function scoreCandidate(source: AnchorCandidate, target: AnchorCandidate) {
  const direction = {
    x: target.point.x - source.point.x,
    y: target.point.y - source.point.y,
  };
  const distance = pointDistance(source.point, target.point);
  const sourceAlignment = dot(normalForSide(source.side), direction);
  const targetAlignment = dot(normalForSide(target.side), {
    x: -direction.x,
    y: -direction.y,
  });
  const sourcePenalty = sourceAlignment > 0 ? 0 : 240;
  const targetPenalty = targetAlignment > 0 ? 0 : 240;

  return distance + sourcePenalty + targetPenalty;
}

/**
 * 返回矩形边的外法线方向。
 */
function normalForSide(side: ConnectorSide): Point {
  switch (side) {
    case "left":
      return { x: -1, y: 0 };
    case "right":
      return { x: 1, y: 0 };
    case "top":
      return { x: 0, y: -1 };
    case "bottom":
      return { x: 0, y: 1 };
  }
}

/**
 * 沿 direction 投射 point。
 */
function project(point: Point, direction: Point, distance: number): Point {
  return {
    x: point.x + direction.x * distance,
    y: point.y + direction.y * distance,
  };
}

/**
 * 计算两个点之间的欧氏距离。
 */
function pointDistance(a: Point, b: Point) {
  return Math.hypot(a.x - b.x, a.y - b.y);
}

/**
 * 点积。
 */
function dot(a: Point, b: Point) {
  return a.x * b.x + a.y * b.y;
}

/**
 * 限制数值范围。
 */
function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

/**
 * SVG path 中的点格式化，减少无意义的小数抖动。
 */
function formatPoint(point: Point) {
  return `${formatNumber(point.x)} ${formatNumber(point.y)}`;
}

/**
 * 将坐标压到 2 位小数。
 */
function formatNumber(value: number) {
  return Number(value.toFixed(2));
}
