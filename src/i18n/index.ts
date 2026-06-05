import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import clipboardEnUS from "@/locales/en-US/clipboard.json";
import commandsEnUS from "@/locales/en-US/commands.json";
import commonEnUS from "@/locales/en-US/common.json";
import preferencesEnUS from "@/locales/en-US/preferences.json";
import previewEnUS from "@/locales/en-US/preview.json";
import clipboardZhCN from "@/locales/zh-CN/clipboard.json";
import commandsZhCN from "@/locales/zh-CN/commands.json";
import commonZhCN from "@/locales/zh-CN/common.json";
import preferencesZhCN from "@/locales/zh-CN/preferences.json";
import previewZhCN from "@/locales/zh-CN/preview.json";
import type { Language } from "@/types/settings";

export const DEFAULT_LANGUAGE: Language = "zh-CN";
export const I18N_NAMESPACES = [
  "common",
  "commands",
  "clipboard",
  "preferences",
  "preview",
] as const;

/**
 * react-i18next 初始化入口；资源来自 JSON，key 使用稳定语义路径。
 */
void i18n.use(initReactI18next).init({
  defaultNS: "common",
  fallbackLng: DEFAULT_LANGUAGE,
  interpolation: {
    escapeValue: false,
  },
  lng: DEFAULT_LANGUAGE,
  ns: I18N_NAMESPACES,
  resources: {
    "en-US": {
      clipboard: clipboardEnUS,
      commands: commandsEnUS,
      common: commonEnUS,
      preferences: preferencesEnUS,
      preview: previewEnUS,
    },
    "zh-CN": {
      clipboard: clipboardZhCN,
      commands: commandsZhCN,
      common: commonZhCN,
      preferences: preferencesZhCN,
      preview: previewZhCN,
    },
  },
});

export default i18n;
