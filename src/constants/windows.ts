/**
 * Tauri 窗口 label 常量。
 */
export const WINDOW_LABEL = {
  /**
   * 自定义右键菜单窗口（仅 Windows，绕开 muda `TrackPopupMenu` 抢焦点的问题）。
   */
  CONTEXT_MENU: "context-menu",
  /**
   * Custom context submenu window on Windows.
   */
  CONTEXT_SUBMENU: "context-submenu",
  /**
   * 主窗口（剪贴板历史列表）。
   */
  MAIN: "main",
  /**
   * 偏好设置窗口。
   */
  PREFERENCE: "preference",
  /**
   * 剪贴板系统级预览 overlay 窗口。
   */
  PREVIEW: "clipboard-preview",
} as const;
