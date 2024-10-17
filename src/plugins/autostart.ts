import { invoke } from "@tauri-apps/api/core";

/**
 * 是否为开机自动启动
 */
export const isAutostart = () => {
	return invoke<boolean>(AUTOSTART_PLUGIN.IS_AUTOSTART);
};
