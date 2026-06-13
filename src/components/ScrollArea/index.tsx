import type { PartialOptions } from "overlayscrollbars";
import { useOverlayScrollbars } from "overlayscrollbars-react";
import type { ComponentPropsWithoutRef, FC } from "react";
import { useEffect, useRef } from "react";
import { cn } from "@/utils/cn";

const SCROLL_AREA_OPTIONS = {
  scrollbars: {
    autoHide: "move",
  },
} satisfies PartialOptions;

type ScrollAreaProps = ComponentPropsWithoutRef<"div">;

/**
 * OverlayScrollbars-backed scroll container for ordinary non-virtual content.
 */
const ScrollArea: FC<ScrollAreaProps> = (props) => {
  const { children, className, ...rest } = props;
  const rootRef = useRef<HTMLDivElement>(null);
  const [initialize, osInstance] = useOverlayScrollbars({
    defer: true,
    options: SCROLL_AREA_OPTIONS,
  });

  useEffect(() => {
    const { current: root } = rootRef;
    if (!root) return;

    initialize(root);

    return () => {
      osInstance()?.destroy();
    };
  }, [initialize, osInstance]);

  return (
    <div
      className={cn("overflow-auto", className)}
      data-overlayscrollbars-initialize=""
      ref={rootRef}
      {...rest}
    >
      {children}
    </div>
  );
};

export default ScrollArea;
