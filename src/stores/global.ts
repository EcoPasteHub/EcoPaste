import { proxy } from "valtio";
import type { GlobalStore } from "@/types/store";
import { DEFAULT_APPEARANCE_GROUP_TABS } from "@/utils/group";

export const globalStore = proxy<GlobalStore>({
  app: {
    autoStart: false,
    showMenubarIcon: true,
    showTaskbarIcon: false,
    silentStart: false,
  },

  appearance: {
    groupTabs: DEFAULT_APPEARANCE_GROUP_TABS.map((item) => ({ ...item })),
    isDark: false,
    theme: "auto",
  },

  env: {},

  integration: {},

  shortcut: {
    clipboard: "Alt+C",
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
