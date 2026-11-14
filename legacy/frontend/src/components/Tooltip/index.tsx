import { Tooltip as AntdTooltip, type TooltipProps } from "antd";
import type { FC } from "react";
import { isValidElement } from "react";

export type OverlayTooltipConfig =
  | TooltipProps["title"]
  | Omit<TooltipProps, "children">;

/**
 * 判断浮层封装收到的是 Tooltip 配置对象，还是直接作为 title 使用的内容。
 */
export const isOverlayTooltipProps = (
  tooltip: OverlayTooltipConfig,
): tooltip is Omit<TooltipProps, "children"> => {
  if (tooltip === null) return false;
  if (Array.isArray(tooltip)) return false;
  if (isValidElement(tooltip)) return false;

  return typeof tooltip === "object";
};

/**
 * 将浮层封装的 tooltip 简写统一转换为 antd Tooltip props。
 */
export const resolveOverlayTooltipProps = (
  tooltip: OverlayTooltipConfig,
): Omit<TooltipProps, "children"> => {
  if (isOverlayTooltipProps(tooltip)) return tooltip;

  return { title: tooltip };
};

/**
 * antd Tooltip 的统一封装：默认开启 `align.overflow.shiftX/Y`，
 * 让浮层贴边时沿轴向平移避开窗口边界（箭头位置不动，仍指向 trigger 中心）。
 * 调用方与原生 Tooltip 完全兼容，传入的 `align` 会整体覆盖默认值。
 */
const Tooltip: FC<TooltipProps> = (props) => {
  const { align, ...rest } = props;

  return (
    <AntdTooltip
      align={
        align ?? { overflow: { adjustY: true, shiftX: true, shiftY: true } }
      }
      {...rest}
    />
  );
};

export default Tooltip;
