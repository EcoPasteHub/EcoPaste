import { useCallback, useEffect, useRef, useState } from "react";

import { useTauriListen } from "@/hooks/useTauriListen";

const NAV_EVENT = "keyboard://nav";

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

  // macOS 路径：Web 键盘事件
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
      dispatch(action);
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [dispatch]);

  // Windows 路径：Rust 钩子 emit
  useTauriListen<NavPayload>(NAV_EVENT, (payload) => {
    dispatch(payload.action);
  });

  return { selectedIndex, setSelectedIndex };
};
