import { invoke } from "@tauri-apps/api/core";

export const COMMAND = {
  PASTE: "plugin:eco-paste|paste",
};

/**
 * 粘贴剪贴板内容
 */
export const paste = () => {
  return invoke(COMMAND.PASTE);
};
