import type { ClipboardPayload, ReadImage, WinOCR } from "@/types/plugin";
import { invoke } from "@tauri-apps/api";
import { listen } from "@tauri-apps/api/event";
import { isEmpty } from "arcdash";

/**
 * 开启监听
 */
export const startListen = async () => {
	invoke(CLIPBOARD_PLUGIN.START_LISTEN);
};

/**
 * 停止监听
 */
export const stopListen = async () => {
	invoke(CLIPBOARD_PLUGIN.STOP_LISTEN);
};

/**
 * 剪切板是否有文件
 */
export const hasFiles = async () => {
	return invoke<boolean>(CLIPBOARD_PLUGIN.HAS_FILES);
};

/**
 * 剪切板是否有图像
 */
export const hasImage = async () => {
	return invoke<boolean>(CLIPBOARD_PLUGIN.HAS_IMAGE);
};

/**
 * 剪切板是否有 HTML 内容
 */
export const hasHTML = async () => {
	return invoke<boolean>(CLIPBOARD_PLUGIN.HAS_HTML);
};

/**
 * 剪切板是否有富文本
 */
export const hasRichText = async () => {
	return invoke<boolean>(CLIPBOARD_PLUGIN.HAS_RICH_TEXT);
};

/**
 * 剪切板是否有纯文本
 */
export const hasText = async () => {
	return invoke<boolean>(CLIPBOARD_PLUGIN.HAS_TEXT);
};

/**
 * 读取剪切板文件
 */
export const readFiles = async (): Promise<ClipboardPayload> => {
	let files = await invoke<string[]>(CLIPBOARD_PLUGIN.READ_FILES);

	files = files.map(decodeURI);

	let size = 0;

	const fileNames = [];

	for await (const path of files) {
		const { size: fileSize, fileName } = await metadata(path);

		size += fileSize;

		fileNames.push(fileName);
	}

	return {
		size,
		search: fileNames.join(" "),
		value: JSON.stringify(files),
	};
};

/**
 * 读取剪切板图片
 */
export const readImage = async (): Promise<ClipboardPayload> => {
	const { saveImageDir } = clipboardStore;

	const { image, ...rest } = await invoke<ReadImage>(
		CLIPBOARD_PLUGIN.READ_IMAGE,
		{ dir: saveImageDir },
	);

	const { size } = await metadata(image);

	let search = await systemOCR(image);

	if (isWin()) {
		const { content, qr } = JSON.parse(search) as WinOCR;

		if (isEmpty(qr)) {
			search = content;
		} else {
			search = qr[0].content;
		}
	}

	const value = image.replace(saveImageDir, "");

	return {
		...rest,
		size,
		search,
		value,
	};
};

/**
 * 读取 HTML 内容
 */
export const readHTML = async (): Promise<ClipboardPayload> => {
	const html = await invoke<string>(CLIPBOARD_PLUGIN.READ_HTML);

	const { value, size } = await readText();

	return {
		size,
		value: html,
		search: value,
	};
};

/**
 * 读取富文本
 */
export const readRichText = async (): Promise<ClipboardPayload> => {
	const richText = await invoke<string>(CLIPBOARD_PLUGIN.READ_RICH_TEXT);

	return {
		value: richText,
		search: richText,
		size: richText.length,
	};
};

/**
 * 读取纯文本
 */
export const readText = async (): Promise<ClipboardPayload> => {
	const text = await invoke<string>(CLIPBOARD_PLUGIN.READ_TEXT);

	return {
		value: text,
		search: text,
		size: text.length,
	};
};

/**
 * 文件写入剪切板
 */
export const writeFiles = (value: string[]) => {
	return invoke(CLIPBOARD_PLUGIN.WRITE_FILES, {
		value,
	});
};

/**
 * 图片写入剪切板
 */
export const writeImage = (value: string) => {
	return invoke(CLIPBOARD_PLUGIN.WRITE_IMAGE, {
		value,
	});
};

/**
 * HTML 内容写入剪切板
 */
export const writeHTML = (text: string, html: string) => {
	return invoke(CLIPBOARD_PLUGIN.WRITE_HTML, {
		text,
		html,
	});
};

/**
 * 富文写入剪切板
 */
export const writeRichText = (value: string) => {
	return invoke(CLIPBOARD_PLUGIN.WRITE_RICH_TEXT, {
		value,
	});
};

/**
 * 纯文本写入剪切板
 */
export const writeText = (value: string) => {
	return invoke(CLIPBOARD_PLUGIN.WRITE_TEXT, {
		value,
	});
};

/**
 * 剪贴板更新
 */
export const onClipboardUpdate = (
	fn: (payload: ClipboardPayload, oldPayload: ClipboardPayload) => void,
) => {
	let payload: ClipboardPayload;
	let oldPayload: ClipboardPayload;

	return listen(CLIPBOARD_PLUGIN.CLIPBOARD_UPDATE, async () => {
		if (await hasFiles()) {
			const filesPayload = await readFiles();

			payload = { ...filesPayload, type: "files" };
		} else if (await hasImage()) {
			const imagePayload = await readImage();

			payload = { ...imagePayload, type: "image" };
		} else if (await hasHTML()) {
			const htmlPayload = await readHTML();

			payload = { ...htmlPayload, type: "html" };
		} else if (await hasRichText()) {
			const richTextPayload = await readRichText();

			payload = { ...richTextPayload, type: "rich-text" };
		} else if (await hasText()) {
			const textPayload = await readText();

			payload = { ...textPayload, type: "text" };
		}

		fn(payload, { ...oldPayload });

		oldPayload = payload;
	});
};
