import { Popover as AntdPopover, type PopoverProps } from "antd";
import type { FC } from "react";

/**
 * antd Popover 的统一封装：默认开启 `align.overflow.shiftX/Y`，
 * 让浮层贴边时沿轴向平移避开窗口边界（箭头位置不动，仍指向 trigger 中心）。
 * 调用方与原生 Popover 完全兼容，传入的 `align` 会整体覆盖默认值。
 */
const Popover: FC<PopoverProps> = (props) => {
  const { align, ...rest } = props;

  return (
    <AntdPopover
      align={
        align ?? { overflow: { adjustY: true, shiftX: true, shiftY: true } }
      }
      {...rest}
    />
  );
};

export default Popover;
