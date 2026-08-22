import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { platform } from "@tauri-apps/plugin-os";
import { WINDOW_LABEL } from "@/constants/windows";
import type { WindowPosition } from "@/types/settings";

/**
 * 当前是否运行在 macOS 平台。
 */
export const isMac = platform() === "macos";

/**
 * 当前是否运行在 Windows 平台。
 */
export const isWin = platform() === "windows";

/**
 * 当前是否为 Vite dev 构建（开发模式）。生产构建为 false。
 */
export const isDev = import.meta.env.DEV;

/**
 * 当前是否为 Windows 平台的剪贴板窗口（focusable=false，需要低级键盘钩子）。
 */
export const isWinClipboardWindow = () => {
  return isWin && getCurrentWebviewWindow().label === WINDOW_LABEL.CLIPBOARD;
};

/**
 * 当前剪贴板窗口是否使用屏幕底部横向布局。
 */
export const isClipboardDockLayout = (position: WindowPosition) => {
  return position === "bottom";
};

/**
 * 判断路径/文件名是否为常见图片类型（按扩展名匹配，大小写不敏感）。
 */
export const isImage = (value: string) => {
  const regex = /\.(jpe?g|png|webp|avif|gif|svg|bmp|ico|tiff?|heic|apng)$/i;

  return regex.test(value);
};
