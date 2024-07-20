import { invoke } from "@tauri-apps/api";

/**
 * 粘贴剪切板内容
 */
export const paste = () => {
	invoke(PASTE_PLUGIN.PASTE);
};
