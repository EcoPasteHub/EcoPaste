import { Tooltip as AntdTooltip, type TooltipProps } from "antd";
import type { FC } from "react";

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
