import type { Language } from "@/types/store";
import type { Locale as AntdLocale } from "antd/es/locale";
import antdEnUS from "antd/locale/en_US";
import antdZhCN from "antd/locale/zh_CN";
import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import enUS from "./en-US.json";
import zhCN from "./zh-CN.json";

i18n.use(initReactI18next).init({
	resources: {
		[LANGUAGE.ZH_CN]: {
			translation: zhCN,
		},
		[LANGUAGE.EN_US]: {
			translation: enUS,
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
		"en-US": antdEnUS,
	};

	return antdLanguage[language];
};
