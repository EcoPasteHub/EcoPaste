import type { Path } from "@/types/router";
import { invoke } from "@tauri-apps/api";
import { appWindow } from "@tauri-apps/api/window";
import { find } from "lodash-es";

/**
 * 创建新窗口
 */
export const createWindow = (path: Path) => {
	const label = path.replace("/", "") ?? "main";

	const options = find(routes, { path })?.meta?.windowOptions;

	invoke("plugin:window|create_window", {
		label,
		options: {
			url: path,
			skipTaskbar: true,
			...options,
		},
	});
};

/**
 * 显示窗口
 */
export const showWindow = () => invoke("plugin:window|show_window");

/**
 * 隐藏窗口
 */
export const hideWindow = () => invoke("plugin:window|hide_window");

/**
 * 退出 app
 */
export const quitApp = () => invoke("plugin:window|quit_app");

/**
 * 切换窗口的显示和隐藏
 */
export const toggleWindowVisible = async () => {
	const focused = await appWindow.isFocused();

	if (focused) {
		hideWindow();
	} else {
		showWindow();
	}
};
