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
