import type { GlobalStore } from "@/types/store";
import { invoke } from "@tauri-apps/api";

/**
 * 获取系统语言
 */
export const getLocale = () => {
	return invoke<GlobalStore["language"]>(LOCALE_PLUGIN.GET_LOCALE);
};
