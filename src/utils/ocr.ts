import { invoke } from "@tauri-apps/api";

export const parse = async (path: string) => {
	return invoke<string>(OCR_PLUGIN.SYSTEM_OCR, { path });
};
