import { WINDOW_PLUGIN } from "@/constants";
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

	invoke(WINDOW_PLUGIN.CREATE_WINDOW, {
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
export const showWindow = () => invoke(WINDOW_PLUGIN.SHOW_WINDOW);

/**
 * 隐藏窗口
 */
export const hideWindow = () => invoke(WINDOW_PLUGIN.HIDE_WINDOW);

/**
 * 退出 app
 */
export const quitApp = () => invoke(WINDOW_PLUGIN.QUIT_APP);

export const noactivate = () => invoke(WINDOW_PLUGIN.NOACTIVATE);

export const activate = () => invoke(WINDOW_PLUGIN.ACTIVATE);

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
