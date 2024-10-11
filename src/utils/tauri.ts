import { ask as tauriAsk } from "@tauri-apps/plugin-dialog";
import {
	type WriteFileOptions,
	remove,
	writeFile as tauriWriteFile,
} from "@tauri-apps/plugin-fs";
import { isObject } from "lodash-es";

/**
 * 询问弹框
 * @param message 弹窗消息
 * @param options 弹窗选项
 */
export const ask: typeof tauriAsk = (message, options) => {
	if (isMac() && isObject(options) && options.kind === "warning") {
		options.kind = "error";
	}

	return tauriAsk(message, options);
};

/**
 *
 * @param path 路径
 * @param data 写入的数据
 * @param options 写入选项
 */
export const writeFile = (
	path: string,
	data: string,
	options?: WriteFileOptions,
) => {
	const encoder = new TextEncoder();
	const encodeData = encoder.encode(data);

	return tauriWriteFile(path, encodeData, options);
};

/**
 * 删除文件
 */
export const removeFile = remove;
