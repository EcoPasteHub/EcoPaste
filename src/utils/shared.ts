import type { Theme } from "@/types/store";
import { invoke } from "@tauri-apps/api";

/**
 * 切换主题
 */
export const toggleTheme = (theme: Theme) => {
	store.theme = theme;

	invoke("plugin:theme|set_theme", { theme });
};
