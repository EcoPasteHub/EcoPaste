import { invoke } from "@tauri-apps/api/core";

export const systemOCR = (path: string) => {
	return invoke<string>(OCR_PLUGIN.SYSTEM_OCR, { path });
};
