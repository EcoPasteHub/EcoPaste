import type { Language } from "@/types/store";
import { invoke } from "@tauri-apps/api/core";
import { values } from "lodash-es";

/**
 * 获取系统语言
 */
export const getLocale = async (): Promise<Language> => {
	const locale = await invoke<Language>(LOCALE_PLUGIN.GET_LOCALE);

	if (values(LANGUAGE).includes(locale)) {
		return locale;
	}

	if (locale.startsWith("zh")) {
		return "zh-TW";
	}

	return "en-US";
};
