import type { ClipboardPayload, ReadImage } from "@/types/plugin";
import { invoke } from "@tauri-apps/api";
import { listen } from "@tauri-apps/api/event";

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
export const readFiles = async () => {
	let files = await invoke<string[]>(CLIPBOARD_PLUGIN.READ_FILES);

	files = files.map(decodeURI);

	let size = 0;

	for await (const path of files) {
		const { size: fileSize } = await metadata(path);

		size += fileSize;
	}

	return {
		size,
		value: files,
	};
};

/**
 * 读取剪切板图片
 */
export const readImage = async () => {
	const { image, ...rest } = await invoke<ReadImage>(
		CLIPBOARD_PLUGIN.READ_IMAGE,
	);

	const { size } = await metadata(image);

	return {
		...rest,
		size,
		value: image,
	};
};

/**
 * 读取 HTML 内容
 */
export const readHTML = async () => {
	const html = await invoke<string>(CLIPBOARD_PLUGIN.READ_HTML);

	const divElement = document.createElement("div");

	divElement.innerHTML = html;

	const size = divElement.innerText.length;

	return {
		size,
		value: html,
	};
};

/**
 * 读取富文本
 */
export const readRichText = async () => {
	const richText = await invoke<string>(CLIPBOARD_PLUGIN.READ_RICH_TEXT);

	return {
		value: richText,
		size: richText.length,
	};
};

/**
 * 读取纯文本
 */
export const readText = async () => {
	const text = await invoke<string>(CLIPBOARD_PLUGIN.READ_TEXT);

	return {
		value: text,
		size: text.length,
	};
};

/**
 * 文件写入剪切板
 */
export const writeFiles = async (value: string[]) => {
	invoke(CLIPBOARD_PLUGIN.WRITE_FILES, {
		value,
	});
};

/**
 * 图片写入剪切板
 */
export const writeImage = async (value: string) => {
	invoke(CLIPBOARD_PLUGIN.WRITE_IMAGE, {
		value,
	});
};

/**
 * HTML 内容写入剪切板
 */
export const writeHTML = async (value: string) => {
	invoke(CLIPBOARD_PLUGIN.WRITE_HTML, {
		value,
	});
};

/**
 * 富文写入剪切板
 */
export const writeRichText = async (value: string) => {
	invoke(CLIPBOARD_PLUGIN.WRITE_RICH_TEXT, {
		value,
	});
};

/**
 * 纯文本写入剪切板
 */
export const writeText = async (value: string) => {
	invoke(CLIPBOARD_PLUGIN.WRITE_TEXT, {
		value,
	});
};

/**
 * 剪贴板更新
 */
export const onClipboardUpdate = (fn: (payload: ClipboardPayload) => void) => {
	let payload: ClipboardPayload;

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

			payload = { ...richTextPayload, type: "rtf" };
		} else if (await hasText()) {
			const textPayload = await readText();

			payload = { ...textPayload, type: "text" };
		}

		fn(payload);
	});
};
