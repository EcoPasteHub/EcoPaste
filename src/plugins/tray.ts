import { invoke } from "@tauri-apps/api";

/**
 * 设置菜单栏图标（托盘图标）的可见性
 */
export const setTrayVisible = (visible: boolean) => {
	invoke(TRAY_PLUGIN.SET_TRAY_VISIBLE, {
		visible,
	});
};
