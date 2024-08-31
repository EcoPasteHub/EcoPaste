import { invoke } from "@tauri-apps/api";

/**
 * 显示和隐藏托盘
 */
export const toggleTrayVisible = (visible: boolean) => {
	invoke(TRAY_PLUGIN.TOGGLE_TRAY_VISIBLE, {
		visible,
	});
};
