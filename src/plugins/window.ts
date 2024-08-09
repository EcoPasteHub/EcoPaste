import { WINDOW_PLUGIN } from "@/constants";
import { invoke } from "@tauri-apps/api";
import { appWindow } from "@tauri-apps/api/window";
import { debounce } from "lodash-es";

/**
 * 显示窗口
 */
export const showWindow = () => {
	invoke(WINDOW_PLUGIN.SHOW_WINDOW);
};

/**
 * 隐藏窗口
 */
export const hideWindow = debounce(
	() => {
		invoke(WINDOW_PLUGIN.HIDE_WINDOW);
	},
	200,
	{
		leading: true,
		trailing: false,
	},
);

/**
 * 给窗口添加阴影
 */
export const setWindowShadow = () => {
	invoke(WINDOW_PLUGIN.SET_WINDOW_SHADOW);
};

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

/**
 * 磨砂窗口
 */
export const frostedWindow = () => {
	invoke(WINDOW_PLUGIN.FROSTED_WINDOW);
};
