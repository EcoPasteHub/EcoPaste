import type { HistoryTablePayload } from "@/types/database";
import type { ClipboardPayload, ReadImage, WindowsOCR } from "@/types/plugin";
import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { exists } from "@tauri-apps/plugin-fs";
import { isEmpty, isEqual } from "lodash-es";

/**
 * 开启监听
 */
export const startListen = () => {
	return invoke(CLIPBOARD_PLUGIN.START_LISTEN);
};

/**
 * 停止监听
 */
export const stopListen = () => {
	return invoke(CLIPBOARD_PLUGIN.STOP_LISTEN);
};

// 切换监听
export const toggleListen = (value: boolean) => {
	if (value) {
		startListen();
	} else {
		stopListen();
	}
};

/**
 * 剪贴板是否有文件
 */
export const hasFiles = () => {
	return invoke<boolean>(CLIPBOARD_PLUGIN.HAS_FILES);
};

/**
 * 剪贴板是否有图像
 */
export const hasImage = () => {
	return invoke<boolean>(CLIPBOARD_PLUGIN.HAS_IMAGE);
};

/**
 * 剪贴板是否有 HTML 内容
 */
export const hasHTML = () => {
	return invoke<boolean>(CLIPBOARD_PLUGIN.HAS_HTML);
};

/**
 * 剪贴板是否有富文本
 */
export const hasRTF = () => {
	return invoke<boolean>(CLIPBOARD_PLUGIN.HAS_RTF);
};

/**
 * 剪贴板是否有纯文本
 */
export const hasText = () => {
	return invoke<boolean>(CLIPBOARD_PLUGIN.HAS_TEXT);
};

/**
 * 读取剪贴板文件
 */
export const readFiles = async (): Promise<ClipboardPayload> => {
	let files = await invoke<string[]>(CLIPBOARD_PLUGIN.READ_FILES);

	files = files.map(decodeURI);

	let count = 0;

	const names = [];

	for await (const path of files) {
		const { size, name } = await metadata(path);

		count += size;

		names.push(name);
	}

	return {
		count,
		search: names.join(" "),
		value: JSON.stringify(files),
		group: "files",
	};
};

/**
 * 读取剪贴板图片
 */
export const readImage = async (): Promise<ClipboardPayload> => {
	const { image, ...rest } = await invoke<ReadImage>(
		CLIPBOARD_PLUGIN.READ_IMAGE,
		{
			dir: getSaveImageDir(),
		},
	);

	const { size: count } = await metadata(image);

	let search = "";

	if (clipboardStore.content.ocr) {
		search = await systemOCR(image);

		if (isWin()) {
			const { content, qr } = JSON.parse(search) as WindowsOCR;

			if (isEmpty(qr)) {
				search = content;
			} else {
				search = qr[0].content;
			}
		}
	}

	const value = image.replace(getSaveImageDir(), "");

	return {
		...rest,
		count,
		value,
		search,
		group: "image",
	};
};

/**
 * 读取 HTML 内容
 */
export const readHTML = async (): Promise<ClipboardPayload> => {
	const html = await invoke<string>(CLIPBOARD_PLUGIN.READ_HTML);

	const { value, count } = await readText();

	return {
		count,
		value: html,
		search: value,
		group: "text",
	};
};

/**
 * 读取富文本
 */
export const readRTF = async (): Promise<ClipboardPayload> => {
	const rtf = await invoke<string>(CLIPBOARD_PLUGIN.READ_RTF);

	const { value, count } = await readText();

	return {
		count,
		value: rtf,
		search: value,
		group: "text",
	};
};

/**
 * 读取纯文本
 */
export const readText = async (): Promise<ClipboardPayload> => {
	const text = await invoke<string>(CLIPBOARD_PLUGIN.READ_TEXT);

	const data: ClipboardPayload = {
		value: text,
		search: text,
		count: text.length,
		group: "text",
	};

	data.subtype = await getClipboardSubtype(data);

	return data;
};

/**
 * 读取剪贴板内容
 */
export const readClipboard = async () => {
	let payload!: ClipboardPayload;

	const { copyPlain } = clipboardStore.content;

	const has = {
		files: await hasFiles(),
		image: await hasImage(),
		html: await hasHTML(),
		rtf: await hasRTF(),
		text: await hasText(),
	};

	if (has.files) {
		const filesPayload = await readFiles();

		payload = { ...filesPayload, type: "files" };
	} else if (has.image && !has.text) {
		const imagePayload = await readImage();

		payload = { ...imagePayload, type: "image" };
	} else if (!copyPlain && has.html) {
		const htmlPayload = await readHTML();

		payload = { ...htmlPayload, type: "html" };
	} else if (!copyPlain && has.rtf) {
		const rtfPayload = await readRTF();

		payload = { ...rtfPayload, type: "rtf" };
	} else {
		const textPayload = await readText();

		payload = { ...textPayload, type: "text" };
	}

	return payload;
};

/**
 * 文件写入剪贴板
 */
export const writeFiles = (value: string) => {
	return invoke(CLIPBOARD_PLUGIN.WRITE_FILES, {
		value: JSON.parse(value),
	});
};

/**
 * 图片写入剪贴板
 */
export const writeImage = (value: string) => {
	return invoke(CLIPBOARD_PLUGIN.WRITE_IMAGE, {
		value,
	});
};

/**
 * HTML 内容写入剪贴板
 */
export const writeHTML = (text: string, html: string) => {
	const { pastePlain } = clipboardStore.content;

	if (pastePlain) {
		return writeText(text);
	}

	return invoke(CLIPBOARD_PLUGIN.WRITE_HTML, {
		text,
		html,
	});
};

/**
 * 富文写入剪贴板
 */
export const writeRTF = (text: string, rtf: string) => {
	const { pastePlain } = clipboardStore.content;

	if (pastePlain) {
		return writeText(text);
	}

	return invoke(CLIPBOARD_PLUGIN.WRITE_RTF, {
		text,
		rtf,
	});
};

/**
 * 纯文本写入剪贴板
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
	// 防抖间隔（ms）
	const DEBOUNCE = 200;
	let lastUpdatedAt = 0;
	let oldPayload: ClipboardPayload;

	return listen(CLIPBOARD_PLUGIN.CLIPBOARD_UPDATE, async () => {
		const payload = await readClipboard();

		if (
			Date.now() - lastUpdatedAt > DEBOUNCE ||
			!isEqual(payload, oldPayload)
		) {
			fn(payload, { ...oldPayload });
		}

		lastUpdatedAt = Date.now();
		oldPayload = payload;
	});
};

/**
 * 将数据写入剪切板
 * @param data 数据
 */
export const writeClipboard = (data?: HistoryTablePayload) => {
	if (!data) return;

	const { type, value, search } = data;

	switch (type) {
		case "text":
			return writeText(value);
		case "rtf":
			return writeRTF(search, value);
		case "html":
			return writeHTML(search, value);
		case "image":
			return writeImage(getSaveImagePath(value));
		case "files":
			return writeFiles(value);
	}
};

/**
 * 粘贴剪切板数据
 * @param data 数据
 * @param plain 是否纯文本粘贴
 */
export const pasteClipboard = async (
	data?: HistoryTablePayload,
	plain = false,
) => {
	if (!data) return;

	const { type, value } = data;

	if (plain) {
		if (type === "files") {
			const pasteValue = JSON.parse(value).join("\n");

			await writeText(pasteValue);
		} else {
			await writeText(data.search);
		}
	} else {
		await writeClipboard(data);
	}

	return paste();
};

/**
 * 获取剪切板数据的子类型
 * @param data 剪贴板数据
 */
export const getClipboardSubtype = async (data: ClipboardPayload) => {
	try {
		const { value } = data;

		let subtype: ClipboardPayload["subtype"];

		if (isURL(value)) {
			subtype = "url";
		} else if (isEmail(value)) {
			subtype = "email";
		} else if (isColor(value)) {
			subtype = "color";
		} else if (await exists(value)) {
			subtype = "path";
		}

		return subtype;
	} catch {
		return;
	}
};
