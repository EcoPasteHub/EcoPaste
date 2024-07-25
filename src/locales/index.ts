import type { Language } from "@/types/store";
import type { Locale as AntdLocale } from "antd/es/locale";
import antdEnUS from "antd/locale/en_US";
import antdJaJP from "antd/locale/ja_JP";
import antdZhCN from "antd/locale/zh_CN";
import antdZhTW from "antd/locale/zh_TW";
import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import enUS from "./en-US.json";
import jaJP from "./ja-JP.json";
import zhCN from "./zh-CN.json";
import zhTW from "./zh-TW.json";

i18n.use(initReactI18next).init({
	resources: {
		[LANGUAGE.ZH_CN]: {
			translation: zhCN,
		},
		[LANGUAGE.ZH_TW]: {
			translation: zhTW,
		},
		[LANGUAGE.EN_US]: {
			translation: enUS,
		},
		[LANGUAGE.JA_JP]: {
			translation: jaJP,
		},
	},
	lng: LANGUAGE.ZH_CN,
	fallbackLng: LANGUAGE.ZH_CN,
	debug: false,
	interpolation: {
		escapeValue: false,
	},
});

export { i18n };

export const getAntdLocale = (language: Language = "zh-CN") => {
	const antdLanguage: Record<Language, AntdLocale> = {
		"zh-CN": antdZhCN,
		"zh-TW": antdZhTW,
		"en-US": antdEnUS,
		"ja-JP": antdJaJP,
	};

	return antdLanguage[language];
};
