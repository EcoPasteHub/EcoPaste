/**
 * Tauri 窗口 label 常量。
 */
export const WINDOW_LABEL = {
  /**
   * 自定义右键菜单窗口（仅 Windows，绕开 muda `TrackPopupMenu` 抢焦点的问题）。
   */
  CONTEXT_MENU: "context-menu",
  /**
   * 主窗口（剪贴板历史列表）。
   */
  MAIN: "main",
  /**
   * 偏好设置窗口。
   */
  PREFERENCE: "preference",
} as const;
