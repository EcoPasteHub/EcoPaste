import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { platform } from "@tauri-apps/plugin-os";
import { WINDOW_LABEL } from "@/constants/windows";

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
 * 当前是否为 Windows 平台的主窗口（focusable=false，需要低级键盘钩子）。
 */
export const isWinMainWindow = () => {
  return isWin && getCurrentWebviewWindow().label === WINDOW_LABEL.MAIN;
};
