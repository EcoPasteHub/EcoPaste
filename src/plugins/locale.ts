import type { Language } from "@/types/store";
import { invoke } from "@tauri-apps/api";
import { values } from "lodash-es";

/**
 * 获取系统语言
 */
export const getLocale = async (): Promise<Language> => {
	const locale = await invoke<Language>(LOCALE_PLUGIN.GET_LOCALE);

	const supportLanguages = values(LANGUAGE);

	if (supportLanguages.includes(locale)) {
		return locale;
	}

	if (locale.startsWith("zh")) {
		return "zh-CN";
	}

	return "en-US";
};

/**
 * 设置语言
 */
export const setLocale = (language: Language = "zh-CN") => {
	invoke(LOCALE_PLUGIN.SET_LOCALE, {
		language,
	});
};
