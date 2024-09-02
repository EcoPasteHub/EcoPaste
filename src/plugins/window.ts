import { WINDOW_PLUGIN } from "@/constants";
import type { WindowLabel } from "@/types/plugin";
import { invoke } from "@tauri-apps/api";
import { emit } from "@tauri-apps/api/event";
import { appWindow } from "@tauri-apps/api/window";
/**
 * 显示窗口
 */
export const showWindow = (label?: WindowLabel) => {
	if (label) {
		emit(LISTEN_KEY.SHOW_WINDOW, label);
	} else {
		invoke(WINDOW_PLUGIN.SHOW_WINDOW);
	}
};

/**
 * 隐藏窗口
 */
export const hideWindow = () => {
	invoke(WINDOW_PLUGIN.HIDE_WINDOW);
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
