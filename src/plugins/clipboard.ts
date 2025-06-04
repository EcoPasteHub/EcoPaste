import type { HistoryTablePayload } from "@/types/database";
import type { ClipboardPayload, ReadImage, WindowsOCR } from "@/types/plugin";
import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { exists } from "@tauri-apps/plugin-fs";
import { isEmpty, isEqual } from "lodash-es";
import { fullName, metadata } from "tauri-plugin-fs-pro-api";

const COMMAND = {
	START_LISTEN: "plugin:eco-clipboard|start_listen",
	STOP_LISTEN: "plugin:eco-clipboard|stop_listen",
	HAS_FILES: "plugin:eco-clipboard|has_files",
	HAS_IMAGE: "plugin:eco-clipboard|has_image",
	HAS_HTML: "plugin:eco-clipboard|has_html",
	HAS_RTF: "plugin:eco-clipboard|has_rtf",
	HAS_TEXT: "plugin:eco-clipboard|has_text",
	READ_FILES: "plugin:eco-clipboard|read_files",
	READ_IMAGE: "plugin:eco-clipboard|read_image",
	READ_HTML: "plugin:eco-clipboard|read_html",
	READ_RTF: "plugin:eco-clipboard|read_rtf",
	READ_TEXT: "plugin:eco-clipboard|read_text",
	WRITE_FILES: "plugin:eco-clipboard|write_files",
	WRITE_IMAGE: "plugin:eco-clipboard|write_image",
	WRITE_HTML: "plugin:eco-clipboard|write_html",
	WRITE_RTF: "plugin:eco-clipboard|write_rtf",
	WRITE_TEXT: "plugin:eco-clipboard|write_text",
	CLIPBOARD_UPDATE: "plugin:eco-clipboard://clipboard_update",
};

/**
 * 开启监听
 */
export const startListen = () => {
	return invoke(COMMAND.START_LISTEN);
};

/**
 * 停止监听
 */
export const stopListen = () => {
	return invoke(COMMAND.STOP_LISTEN);
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
	return invoke<boolean>(COMMAND.HAS_FILES);
};

/**
 * 剪贴板是否有图像
 */
export const hasImage = () => {
	return invoke<boolean>(COMMAND.HAS_IMAGE);
};

/**
 * 剪贴板是否有 HTML 内容
 */
export const hasHTML = () => {
	return invoke<boolean>(COMMAND.HAS_HTML);
};

/**
 * 剪贴板是否有富文本
 */
export const hasRTF = () => {
	return invoke<boolean>(COMMAND.HAS_RTF);
};

/**
 * 剪贴板是否有纯文本
 */
export const hasText = () => {
	return invoke<boolean>(COMMAND.HAS_TEXT);
};

/**
 * 读取剪贴板文件
 */
export const readFiles = async (): Promise<ClipboardPayload> => {
	let files = await invoke<string[]>(COMMAND.READ_FILES);

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
	const { image, ...rest } = await invoke<ReadImage>(COMMAND.READ_IMAGE, {
		path: getSaveImagePath(),
	});

	const { size: count } = await metadata(image);

	let search = "";

	if (clipboardStore.content.ocr) {
		search = await systemOCR(image);

		if (isWin) {
			const { content, qr } = JSON.parse(search) as WindowsOCR;

			if (isEmpty(qr)) {
				search = content;
			} else {
				search = qr[0].content;
			}
		}
	}

	const value = await fullName(image);

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
	const html = await invoke<string>(COMMAND.READ_HTML);

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
	const rtf = await invoke<string>(COMMAND.READ_RTF);

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
	const text = await invoke<string>(COMMAND.READ_TEXT);

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
 * 文件写入剪贴板
 */
export const writeFiles = (value: string) => {
	return invoke(COMMAND.WRITE_FILES, {
		value: JSON.parse(value),
	});
};

/**
 * 图片写入剪贴板
 */
export const writeImage = (value: string) => {
	return invoke(COMMAND.WRITE_IMAGE, {
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

	return invoke(COMMAND.WRITE_HTML, {
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

	return invoke(COMMAND.WRITE_RTF, {
		text,
		rtf,
	});
};

/**
 * 纯文本写入剪贴板
 */
export const writeText = (value: string) => {
	return invoke(COMMAND.WRITE_TEXT, {
		value,
	});
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
 * 剪贴板更新
 */
export const onClipboardUpdate = (fn: (payload: ClipboardPayload) => void) => {
	let lastUpdated = 0;
	let previousPayload: ClipboardPayload;

	return listen(COMMAND.CLIPBOARD_UPDATE, async () => {
		const payload = await readClipboard();

		const { group, count } = payload;

		if (group === "text" && count === 0) {
			return;
		}

		const expired = Date.now() - lastUpdated > 200;

		if (expired || !isEqual(payload, previousPayload)) {
			fn(payload);
		}

		lastUpdated = Date.now();
		previousPayload = payload;
	});
};

/**
 * 将数据写入剪贴板
 * @param data 数据
 */
export const writeClipboard = (data?: HistoryTablePayload) => {
	if (!data) return;

	const { type, value, search } = data;

	// 写入剪贴板后隐藏窗口
	hideWindow();

	switch (type) {
		case "text":
			return writeText(value);
		case "rtf":
			return writeRTF(search, value);
		case "html":
			return writeHTML(search, value);
		case "image":
			return writeImage(resolveImagePath(value));
		case "files":
			return writeFiles(value);
	}
};

/**
 * 粘贴剪贴板数据
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
 * 获取剪贴板数据的子类型
 * @param data 剪贴板数据
 */
export const getClipboardSubtype = async (data: ClipboardPayload) => {
	try {
		const { value } = data;

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

/**
 * 将多种格式的数据写入剪贴板
 * @param items 多个剪贴板数据项
 */
export const writeMultipleClipboard = async (items: HistoryTablePayload[]) => {
	if (!items || items.length === 0) return;

	// 按类型分组处理
	const textItems = items.filter((item) => item.type === "text");
	const htmlItems = items.filter((item) => item.type === "html");

	// 处理文本内容
	if (textItems.length > 0) {
		const combinedText = textItems.map((item) => item.value).join("\n");
		await writeText(combinedText);
	}

	// 处理HTML内容
	if (htmlItems.length > 0) {
		const combinedHtml = htmlItems.map((item) => item.value).join("\n");
		const combinedText = htmlItems.map((item) => item.search).join("\n");
		await writeHTML(combinedText, combinedHtml);
	}

	// 写入剪贴板后隐藏窗口
	hideWindow();
};
