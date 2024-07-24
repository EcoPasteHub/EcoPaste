import { invoke } from "@tauri-apps/api";

/**
 * 判断是否是开机自动启动
 */
export const isAutoLaunch = () => {
	return invoke<boolean>(AUTO_LAUNCH_PLUGIN.IS_AUTO_LAUNCH);
};
