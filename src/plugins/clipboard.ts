import type { DatabaseSchemaHistory } from "@/types/database";
import { exists } from "@tauri-apps/plugin-fs";
import {
	writeFiles,
	writeHTML,
	writeImage,
	writeRTF,
	writeText,
} from "tauri-plugin-clipboard-x-api";

export const getClipboardTextSubtype = async (value: string) => {
	try {
		if (isURL(value)) {
			return "url";
		}

		if (isEmail(value)) {
			return "email";
		}

		if (isColor(value)) {
			return "color";
		}

		if (await exists(value)) {
			return "path";
		}
	} catch {
		return;
	}
};

export const writeToClipboard = (data: DatabaseSchemaHistory) => {
	const { type, value, search } = data;

	switch (type) {
		case "text":
			return writeText(value);
		case "rtf":
			return writeRTF(search, value);
		case "html":
			return writeHTML(search, value);
		case "image":
			return writeImage(value);
		case "files":
			return writeFiles(value);
	}
};

export const pasteToClipboard = async (
	data: DatabaseSchemaHistory,
	asText = false,
) => {
	const { type, value, search } = data;

	if (asText) {
		if (type === "files") {
			await writeText(value.join("\n"));
		} else {
			await writeText(search);
		}
	} else {
		await writeToClipboard(data);
	}

	return paste();
};
