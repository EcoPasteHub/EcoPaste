import type { WindowPosition } from "@/types/settings";

/**
 * 判断主窗口是否使用横向面板排版；当前所有弹出位置共享这套视觉布局。
 */
export function usesClipboardSheetLayout(position: WindowPosition): boolean {
  switch (position) {
    case "bottomSheet":
    case "center":
    case "followCursor":
    case "remember":
      return true;
    default: {
      const exhaustive: never = position;

      return exhaustive;
    }
  }
}

export function isClipboardBottomSheet(position: WindowPosition): boolean {
  return position === "bottomSheet";
}
