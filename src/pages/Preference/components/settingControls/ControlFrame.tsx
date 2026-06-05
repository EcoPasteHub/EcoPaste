import type { FC, ReactNode } from "react";
import { cn } from "@/utils/cn";

interface ControlFrameProps {
  children: ReactNode;
  className?: string;
}

/**
 * 统一右侧控制区的可视节奏，让不同控件共享同一命中区域。
 */
const ControlFrame: FC<ControlFrameProps> = (props) => {
  const { children, className } = props;

  return (
    <span className={cn("flex h-8 w-28 items-center justify-end", className)}>
      {children}
    </span>
  );
};

export default ControlFrame;
