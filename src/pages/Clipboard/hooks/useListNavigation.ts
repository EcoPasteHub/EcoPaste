import { useCallback, useEffect, useRef, useState } from "react";
import { TAURI_EVENT } from "@/constants/events";
import { useTauriListen } from "@/hooks/useTauriListen";

type NavAction = "up" | "down" | "enter" | "escape";

interface NavPayload {
  action: NavAction;
}

interface UseListNavigationOptions {
  count: number;
  onEnter: (index: number) => void;
  onEscape: () => void;
}

interface UseListNavigationResult {
  selectedIndex: number;
  setSelectedIndex: (index: number) => void;
}

// 两路同时挂：macOS 走 window keydown；Windows focusable=false 时 WebView 收不到键，
// 走 Rust 钩子 emit 的 `keyboard://nav`。另一端自然不会触发，不做平台分支。
export const useListNavigation = ({
  count,
  onEnter,
  onEscape,
}: UseListNavigationOptions): UseListNavigationResult => {
  const [selectedIndex, setSelectedIndexState] = useState(0);
  const selectedRef = useRef(selectedIndex);
  selectedRef.current = selectedIndex;
  const countRef = useRef(count);
  countRef.current = count;
  const onEnterRef = useRef(onEnter);
  onEnterRef.current = onEnter;
  const onEscapeRef = useRef(onEscape);
  onEscapeRef.current = onEscape;

  // count 变化时把 selectedIndex 夹回有效区间，避免删条目后越界。
  useEffect(() => {
    setSelectedIndexState((prev) => {
      if (count === 0) return 0;
      if (prev >= count) return count - 1;
      return prev;
    });
  }, [count]);

  const setSelectedIndex = useCallback((index: number) => {
    const c = countRef.current;
    if (c === 0) return;
    setSelectedIndexState(Math.max(0, Math.min(index, c - 1)));
  }, []);

  const dispatch = useCallback((action: NavAction) => {
    const c = countRef.current;
    if (c === 0 && action !== "escape") return;
    const cur = selectedRef.current;
    switch (action) {
      case "up":
        setSelectedIndexState(cur > 0 ? cur - 1 : 0);
        break;
      case "down":
        setSelectedIndexState(cur < c - 1 ? cur + 1 : c - 1);
        break;
      case "enter":
        onEnterRef.current(cur);
        break;
      case "escape":
        onEscapeRef.current();
        break;
    }
  }, []);

  // macOS 路径：Web 键盘事件。
  // 用 capture 阶段 + stopPropagation：搜索框聚焦时，react-aria 的 SearchField 也
  // 监听 Enter/Escape（Enter 触发 submit、Escape 清空输入框），bubble 阶段才到 window
  // 就被它先处理了。在 capture 阶段拦住，让上/下/回车/Esc 一律直达列表导航，
  // 而不是被输入框「拦截」。preventDefault 抑制默认行为（form submit / 光标移动等）。
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      let action: NavAction | null = null;
      switch (e.key) {
        case "ArrowUp":
          action = "up";
          break;
        case "ArrowDown":
          action = "down";
          break;
        case "Enter":
          action = "enter";
          break;
        case "Escape":
          action = "escape";
          break;
      }
      if (!action) return;
      e.preventDefault();
      e.stopPropagation();
      dispatch(action);
    };
    window.addEventListener("keydown", handler, true);
    return () => window.removeEventListener("keydown", handler, true);
  }, [dispatch]);

  // Windows 路径：Rust 钩子 emit
  useTauriListen<NavPayload>(TAURI_EVENT.KEYBOARD_NAV, (payload) => {
    dispatch(payload.action);
  });

  return { selectedIndex, setSelectedIndex };
};
