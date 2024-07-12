import type { Theme } from "@/types/store";
import { invoke } from "@tauri-apps/api";

/**
 * 获取主题色
 */
export const getTheme = () => {
	return invoke<Theme>(THEME_PLUGIN.GET_THEME);
};

/**
 * 设置主题色
 * @param theme 主题色
 */
export const setTheme = (theme: Theme) => {
	invoke(THEME_PLUGIN.SET_THEME, { theme });
};
