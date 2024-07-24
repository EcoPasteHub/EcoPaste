import { invoke } from "@tauri-apps/api";

export const isAutoLaunch = () => {
	return invoke<boolean>(AUTO_LAUNCH_PLUGIN.IS_AUTO_LAUNCH);
};
