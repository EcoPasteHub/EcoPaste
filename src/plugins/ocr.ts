import { invoke } from "@tauri-apps/api/core";

const COMMAND = {
	SYSTEM_OCR: "plugin:eco-ocr|system_ocr",
};

export const systemOCR = (path: string) => {
	return invoke<string>(COMMAND.SYSTEM_OCR, { path });
};
