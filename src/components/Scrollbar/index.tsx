import { MacScrollbar, type MacScrollbarProps } from "mac-scrollbar";
import { forwardRef, useImperativeHandle, useRef } from "react";
import { useSnapshot } from "valtio";
import { globalStore } from "@/stores/global";

interface ScrollbarProps extends MacScrollbarProps {
  thumbSize?: number;
  offset?: number;
}

const Scrollbar = forwardRef<HTMLElement, ScrollbarProps>((props, ref) => {
  const { appearance } = useSnapshot(globalStore);

  const { thumbSize = 6, offset = 0, children, ...rest } = props;

  const containerRef = useRef<HTMLElement>(null);

  useImperativeHandle(ref, () => containerRef.current!);

  const getThumbStyle: MacScrollbarProps["thumbStyle"] = (horizontal) => {
    if (horizontal) {
      return {
        bottom: offset,
        height: thumbSize,
      };
    }

    return {
      right: offset,
      width: thumbSize,
    };
  };

  return (
    <MacScrollbar
      {...rest}
      ref={containerRef}
      skin={appearance.isDark ? "dark" : "light"}
      thumbStyle={getThumbStyle}
      trackStyle={() => ({ "--ms-track-size": 0, border: 0 })}
    >
      {children}
    </MacScrollbar>
  );
});

export default Scrollbar;
