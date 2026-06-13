import type { PartialOptions } from "overlayscrollbars";
import {
  OverlayScrollbarsComponent,
  type OverlayScrollbarsComponentRef,
} from "overlayscrollbars-react";
import type { ComponentPropsWithoutRef, FC, Ref } from "react";
import { useImperativeHandle, useRef, useState } from "react";
import { cn } from "@/utils/cn";

const SCROLL_AREA_OPTIONS = {
  scrollbars: {
    autoHide: "move",
  },
} satisfies PartialOptions;

type ScrollAreaProps = ComponentPropsWithoutRef<"div"> & {
  contentClassName?: string;
  ref?: Ref<HTMLDivElement>;
};

/**
 * OverlayScrollbars-backed scroll container for ordinary non-virtual content.
 */
const ScrollArea: FC<ScrollAreaProps> = (props) => {
  const { children, className, contentClassName, ref, ...rest } = props;
  const componentRef = useRef<OverlayScrollbarsComponentRef<"div">>(null);
  const [viewport, setViewport] = useState<HTMLElement | null>(null);

  useImperativeHandle(ref, () => {
    return (viewport ?? componentRef.current?.getElement()) as HTMLDivElement;
  }, [viewport]);

  return (
    <OverlayScrollbarsComponent
      className={cn("overflow-auto", className)}
      defer
      events={{
        initialized(instance) {
          setViewport(instance.elements().viewport);
        },
      }}
      options={SCROLL_AREA_OPTIONS}
      ref={componentRef}
      {...rest}
    >
      {contentClassName ? (
        <div className={contentClassName}>{children}</div>
      ) : (
        children
      )}
    </OverlayScrollbarsComponent>
  );
};

export default ScrollArea;
