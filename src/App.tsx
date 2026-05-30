// HeroUI v3 不需要 Provider 包裹（它读取 html 上的 class/data-theme + CSS 变量）。
// 主题切换见 useApplyTheme；i18n 切换见 useApplyLanguage（初始化在 main.tsx）。

import { RouterProvider } from "react-router";
import { useApplyLanguage } from "@/hooks/useApplyLanguage";
import { useApplyTheme } from "@/hooks/useApplyTheme";
import { router } from "@/router";

const App = () => {
  useApplyTheme();
  useApplyLanguage();
  return <RouterProvider router={router} />;
};

export default App;
