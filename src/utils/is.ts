// 平台与内容类型判断的统一出口。
// 平台判断走 @tauri-apps/plugin-os 的 `platform()`，结果由 Rust 在启动时确定后注入，可同步调用。

import { platform } from "@tauri-apps/plugin-os";

/**
 * 当前是否运行在 macOS 平台。
 */
export const isMac = platform() === "macos";

/**
 * 当前是否运行在 Windows 平台。
 */
export const isWin = platform() === "windows";

/**
 * 是否为图片路径或文件名（按扩展名判断，大小写不敏感）。
 */
export const isImage = (value: string) => {
  const regex = /\.(jpe?g|png|webp|avif|gif|svg|bmp|ico|tiff?|heic|apng)$/i;

  return regex.test(value);
};
