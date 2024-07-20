import type { Language } from "@/types/store";
import { invoke } from "@tauri-apps/api";

/**
 * 获取系统语言
 */
export const getLocale = () => {
	return invoke<Language>(LOCALE_PLUGIN.GET_LOCALE);
};

/**
 * 设置语言
 */
export const setLocale = (language?: Language) => {
	invoke(LOCALE_PLUGIN.SET_LOCALE, {
		language,
	});
};
