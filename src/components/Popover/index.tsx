import { Popover as AntdPopover, type PopoverProps } from "antd";
import type { FC } from "react";
import { useState } from "react";
import Tooltip, {
  type OverlayTooltipConfig,
  resolveOverlayTooltipProps,
} from "@/components/Tooltip";

export interface AppPopoverProps extends PopoverProps {
  tooltip?: OverlayTooltipConfig | false;
}

/**
 * 包装触发节点 Tooltip，并在 Popover 打开时强制收起 Tooltip。
 */
const renderPopoverTrigger = (
  children: PopoverProps["children"],
  tooltip: OverlayTooltipConfig | false | undefined,
  open: boolean,
): PopoverProps["children"] => {
  if (tooltip === false || tooltip === null || tooltip === void 0) {
    return children;
  }

  const tooltipProps = resolveOverlayTooltipProps(tooltip);

  return (
    <Tooltip {...tooltipProps} open={open ? false : tooltipProps.open}>
      {children}
    </Tooltip>
  );
};

/**
 * antd Popover 的统一封装：默认开启 `align.overflow.shiftX/Y`，
 * 让浮层贴边时沿轴向平移避开窗口边界（箭头位置不动，仍指向 trigger 中心）。
 * 调用方与原生 Popover 完全兼容，传入的 `align` 会整体覆盖默认值。
 */
const Popover: FC<AppPopoverProps> = (props) => {
  const { align, children, onOpenChange, open, tooltip, ...rest } = props;
  const [innerOpen, setInnerOpen] = useState(false);
  const mergedOpen = open ?? innerOpen;

  const handleOpenChange: NonNullable<PopoverProps["onOpenChange"]> = (
    nextOpen,
  ) => {
    setInnerOpen(nextOpen);
    onOpenChange?.(nextOpen);
  };

  return (
    <AntdPopover
      align={
        align ?? { overflow: { adjustY: true, shiftX: true, shiftY: true } }
      }
      onOpenChange={handleOpenChange}
      open={open}
      {...rest}
    >
      {renderPopoverTrigger(children, tooltip, mergedOpen)}
    </AntdPopover>
  );
};

export default Popover;
