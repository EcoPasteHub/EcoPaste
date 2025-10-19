import type { Locale as AntdLocale } from "antd/es/locale";
import antdEnUS from "antd/locale/en_US";
import antdJaJP from "antd/locale/ja_JP";
import antdZhCN from "antd/locale/zh_CN";
import antdZhTW from "antd/locale/zh_TW";
import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import type { Language } from "@/types/store";
import enUS from "./en-US.json";
import jaJP from "./ja-JP.json";
import zhCN from "./zh-CN.json";
import zhTW from "./zh-TW.json";

i18n.use(initReactI18next).init({
  debug: false,
  fallbackLng: LANGUAGE.ZH_CN,
  interpolation: {
    escapeValue: false,
  },
  lng: LANGUAGE.ZH_CN,
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
});

export { i18n };

export const getAntdLocale = (language: Language = LANGUAGE.ZH_CN) => {
  const antdLanguage: Record<Language, AntdLocale> = {
    [LANGUAGE.ZH_CN]: antdZhCN,
    [LANGUAGE.ZH_TW]: antdZhTW,
    [LANGUAGE.EN_US]: antdEnUS,
    [LANGUAGE.JA_JP]: antdJaJP,
  };

  return antdLanguage[language];
};
