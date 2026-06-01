import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";

/**
 * 获取当前 webview 窗口实例。
 */
export const currentWindow = getCurrentWebviewWindow();
