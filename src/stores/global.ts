import { proxy } from "valtio";
import type { GlobalStore } from "@/types/store";

export const globalStore = proxy<GlobalStore>({
  app: {
    autoStart: false,
    showMenubarIcon: true,
    showTaskbarIcon: false,
    silentStart: false,
  },

  appearance: {
    isDark: false,
    theme: "auto",
  },

  env: {},

  ocr: {
    apiBase: "https://api.openai.com",
    apiKey: "",
    autoCopy: true,
    model: "gpt-4o",
    translate: {
      apiBase: "",
      apiKey: "",
      enabled: true,
      model: "",
      targetLanguage: "zh-CN",
    },
  },

  shortcut: {
    clipboard: "Alt+C",
    ocr: "Alt+O",
    ocrTranslate: "Alt+T",
    pastePlain: "",
    preference: "Alt+X",
    quickPaste: {
      enable: false,
      value: "Command+Shift",
    },
  },

  update: {
    auto: false,
    beta: false,
  },
});
