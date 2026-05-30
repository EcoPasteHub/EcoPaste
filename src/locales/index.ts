// i18next 初始化。目前仅支持 zh-CN / en-US。

import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import type { Language } from "@/types/settings";
import enUS from "./en-US.json";
import zhCN from "./zh-CN.json";

const resources = {
  "en-US": { translation: enUS },
  "zh-CN": { translation: zhCN },
} as const;

export async function initI18n(language: Language): Promise<void> {
  if (i18n.isInitialized) {
    if (i18n.language !== language) await i18n.changeLanguage(language);
    return;
  }
  await i18n.use(initReactI18next).init({
    fallbackLng: "zh-CN",
    interpolation: { escapeValue: false },
    lng: language,
    resources,
    returnNull: false,
  });
}

export default i18n;
