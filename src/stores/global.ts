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
    autoSpeak: false,
    hideMainWindow: true,
    model: "gpt-4o",
    prompt:
      "请识别图片中的所有文字，只输出识别到的文字内容，不要添加任何解释、格式或标点符号修改。如果图片中没有文字，请回复[无文字]。",
    saveHistory: true,
    translate: {
      apiBase: "",
      apiKey: "",
      enabled: true,
      model: "",
      systemPrompt:
        "You are a professional translator. Translate the following text to $targetLanguage. Only output the translation result, without any explanations, notes, or the original text.",
      targetLanguage: "zh-CN",
    },
    windowPinned: false,
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
