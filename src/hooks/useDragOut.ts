import type { MouseEvent as ReactMouseEvent } from "react";
import { useRef } from "react";
import { startDragClipboardItem } from "@/commands";

/**
 * 拖拽阈值（像素）：mousedown 后位移超过该值才触发 OS 级 drag，避免点击 / 文本选中误触发。
 */
const DRAG_THRESHOLD = 5;

/**
 * 列表项 drag-out 检测 hook：返回挂到目标节点 `onMouseDown` 的回调。
 *
 * 行为：
 * 1. 左键 mousedown 记录起点，挂 mousemove / mouseup 到 window。
 * 2. mousemove 累计位移 >= 阈值 → 调用 Rust `start_drag_clipboard_item`，
 *    随后清理事件（OS 接管光标后浏览器侧的 mousemove 不再可靠）。
 * 3. mouseup 触发前未越过阈值 → 视为点击，仅清理事件，不调命令。
 *
 * 不阻止 mousedown / click 事件冒泡：上层的「悬停选中」/「右键菜单」仍正常工作。
 */
export const useDragOut = (itemId: string) => {
  // 用 ref 持有当前 itemId，避免每次 render 重新 attach mousedown handler；
  // 同时 mousemove 闭包能拿到最新 id（itemId 变化时下一次 mousedown 自然取新值）。
  const idRef = useRef(itemId);
  idRef.current = itemId;

  const handleMouseDown = (event: ReactMouseEvent) => {
    if (event.button !== 0) return;

    const startX = event.clientX;
    const startY = event.clientY;
    let triggered = false;

    const cleanup = () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", cleanup);
    };

    const handleMove = async (move: MouseEvent) => {
      if (triggered) return;

      const dx = move.clientX - startX;
      const dy = move.clientY - startY;
      if (Math.hypot(dx, dy) < DRAG_THRESHOLD) return;

      triggered = true;
      cleanup();

      try {
        await startDragClipboardItem(idRef.current);
      } catch {
        // 错误 toast 已在 commands/index.ts 内统一处理。
      }
    };

    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", cleanup);
  };

  return { onMouseDown: handleMouseDown };
};
