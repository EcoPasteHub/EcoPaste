import { ask as tauriAsk } from "@tauri-apps/plugin-dialog";
import { remove } from "@tauri-apps/plugin-fs";
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
 * 删除文件
 */
export const removeFile = remove;
