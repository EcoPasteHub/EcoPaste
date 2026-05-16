import { MacScrollbar, type MacScrollbarProps } from "mac-scrollbar";
import { forwardRef, useImperativeHandle, useRef } from "react";
import { useSnapshot } from "valtio";
import { globalStore } from "@/stores/global";

interface ScrollbarProps extends MacScrollbarProps {
  thumbSize?: number;
  offsetX?: number;
  offsetY?: number;
}

const Scrollbar = forwardRef<HTMLElement, ScrollbarProps>((props, ref) => {
  const { appearance } = useSnapshot(globalStore);

  const { thumbSize = 6, offsetX = 0, offsetY = 0, children, ...rest } = props;

  const containerRef = useRef<HTMLElement>(null);

  useImperativeHandle(ref, () => containerRef.current!);

  const getThumbStyle: MacScrollbarProps["thumbStyle"] = (horizontal) => {
    if (horizontal) {
      return {
        bottom: offsetY,
        height: thumbSize,
      };
    }

    return {
      right: offsetX,
      width: thumbSize,
    };
  };

  const getTrackStyle: MacScrollbarProps["trackStyle"] = () => {
    return {
      "--ms-track-size": 0,
      border: 0,
    };
  };

  return (
    <MacScrollbar
      {...rest}
      ref={containerRef}
      skin={appearance.isDark ? "dark" : "light"}
      thumbStyle={getThumbStyle}
      trackStyle={getTrackStyle}
    >
      {children}
    </MacScrollbar>
  );
});

export default Scrollbar;
