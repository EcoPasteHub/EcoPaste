import { invoke } from "@tauri-apps/api/core";

/**
 * 粘贴剪贴板内容
 */
export const paste = async () => {
	return await invoke(PASTE_PLUGIN.PASTE);
};
