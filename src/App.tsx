// HeroUI v3 不需要 Provider 包裹（它读取 html 上的 class/data-theme + CSS 变量）。
// 主题切换见 useApplyTheme；i18n 切换见 useApplyLanguage（初始化在 main.tsx）。

import { RouterProvider } from "react-router";
import { TAURI_EVENT } from "@/constants/events";
import { useApplyLanguage } from "@/hooks/useApplyLanguage";
import { useApplyTheme } from "@/hooks/useApplyTheme";
import { useTauriListen } from "@/hooks/useTauriListen";
import { router } from "@/router";
import { applySettings } from "@/stores/settings";
import type { Settings } from "@/types/settings";

const App = () => {
  useApplyTheme();
  useApplyLanguage();
  useTauriListen<Settings>(TAURI_EVENT.SETTINGS_UPDATED, applySettings);
  return <RouterProvider router={router} />;
};

export default App;
