import { useLatest } from "ahooks";
import { type FocusEvent, useCallback } from "react";
import {
  focusMainWindowForTextInput,
  restoreMainWindowNonFocusable,
} from "@/commands";
import { isWinMainWindow } from "@/utils/is";
import { log } from "@/utils/log";

interface UseMainWindowTextInputFocusOptions<Element extends HTMLElement> {
  onBlur?: (event: FocusEvent<Element>) => void;
  onFocus?: (event: FocusEvent<Element>) => void;
}

/**
 * Windows 主窗口平时 `focusable=false`，输入框编辑时需要临时切到可聚焦窗口来接收字符输入。
 */
export const useMainWindowTextInputFocus = <Element extends HTMLElement>(
  options: UseMainWindowTextInputFocusOptions<Element>,
) => {
  const { onBlur, onFocus } = options;
  const onBlurRef = useLatest(onBlur);
  const onFocusRef = useLatest(onFocus);
  const needsWindowFocusMode = isWinMainWindow();

  /**
   * Windows text editing must acquire OS focus before the DOM input can receive characters.
   */
  const focusWindowForTextInput = useCallback(async () => {
    if (!needsWindowFocusMode) return;

    try {
      await focusMainWindowForTextInput();
    } catch (error) {
      log.error("focus main window for text input failed", error);
    }
  }, [needsWindowFocusMode]);

  /**
   * 输入框获得 DOM 焦点时同步获取系统键盘焦点，否则 Windows 会把文字送给原前台窗口。
   */
  const handleFocus = async (event: FocusEvent<Element>) => {
    await focusWindowForTextInput();

    onFocusRef.current?.(event);
  };

  /**
   * 输入框退出编辑时恢复主窗口不抢焦点模式，并重新启用 Windows 导航键钩子。
   */
  const handleBlur = async (event: FocusEvent<Element>) => {
    onBlurRef.current?.(event);

    if (!needsWindowFocusMode) return;

    try {
      await restoreMainWindowNonFocusable();
    } catch (error) {
      log.error("restore main window focus mode failed", error);
    }
  };

  return {
    focusWindowForTextInput,
    onBlur: handleBlur,
    onFocus: handleFocus,
  };
};
