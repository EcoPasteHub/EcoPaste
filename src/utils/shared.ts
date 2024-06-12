import type { Theme } from "@/types/store";
import { invoke } from "@tauri-apps/api";
import { message } from "@tauri-apps/api/dialog";
import { type } from "@tauri-apps/api/os";

/**
 * 切换主题
 */
export const toggleTheme = async (theme: Theme) => {
	const osType = await type();

	if (osType === "Windows_NT") {
		await message("切换主题需要重启 app 才能生效！", {
			okLabel: "重启",
		});
	}

	globalStore.theme = theme;

	invoke("plugin:theme|set_theme", { theme });
};

/**
 * 是否为开发环境
 */
export const isDev = () => {
	return import.meta.env.DEV;
};

/**
 * 是否为 windows 系统
 */
export const isWin = async () => {
	const osType = await type();

	return osType === "Windows_NT";
};

/**
 * 是否为 mac 系统
 */
export const isMac = async () => {
	const osType = await type();

	return osType === "Darwin";
};
