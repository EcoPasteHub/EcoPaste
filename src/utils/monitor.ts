import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { cursorPosition, monitorFromPoint } from "@tauri-apps/api/window";

/**
 * 获取当前鼠标所在的显示器
 */
export const getCursorMonitor = async () => {
  const appWindow = getCurrentWebviewWindow();
  const scaleFactor = await appWindow.scaleFactor();

  const cursorPoint = await cursorPosition();
  const { x, y } = cursorPoint.toLogical(scaleFactor);

  const monitor = await monitorFromPoint(x, y);

  if (!monitor) return;

  return {
    ...monitor,
    cursorPoint,
  };
};
