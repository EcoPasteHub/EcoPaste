import { invoke } from "@tauri-apps/api/core";

/**
 * 粘贴剪贴板内容
 */
export const paste = () => {
	return invoke(PASTE_PLUGIN.PASTE);
};
