import type { Language } from "@/types/store";
import type { Locale as AntdLocale } from "antd/es/locale";
import antdEnUS from "antd/locale/en_US";
import antdJaJP from "antd/locale/ja_JP";
import antdViVN from "antd/locale/vi_VN";
import antdZhCN from "antd/locale/zh_CN";
import antdZhTW from "antd/locale/zh_TW";
import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import enUS from "./en-US.json";
import jaJP from "./ja-JP.json";
import viVN from "./vi-VN.json";
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
		[LANGUAGE.VI_VN]: {
			translation: viVN,
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

export const getAntdLocale = (language: Language = LANGUAGE.ZH_CN) => {
	const antdLanguage: Record<Language, AntdLocale> = {
		[LANGUAGE.ZH_CN]: antdZhCN,
		[LANGUAGE.ZH_TW]: antdZhTW,
		[LANGUAGE.EN_US]: antdEnUS,
		[LANGUAGE.JA_JP]: antdJaJP,
		[LANGUAGE.VI_VN]: antdViVN,
	};

	return antdLanguage[language];
};
