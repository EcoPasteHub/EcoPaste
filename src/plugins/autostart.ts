import { invoke } from "@tauri-apps/api/core";

const COMMAND = {
  IS_AUTOSTART: "plugin:eco-autostart|is_autostart",
};

/**
 * 是否为开机自动启动
 */
export const isAutostart = () => {
  return invoke<boolean>(COMMAND.IS_AUTOSTART);
};
