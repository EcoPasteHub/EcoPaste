// HeroUI v3 不需要 Provider 包裹（它读取 html 上的 class/data-theme + CSS 变量）。
// 主题切换（跟随系统/明暗）见 7.1.3；i18n 见 7.5——届时在此处插入对应 Provider/Hook。

import { RouterProvider } from "react-router";
import { useApplyTheme } from "@/hooks/useApplyTheme";
import { router } from "@/router";

const App = () => {
  useApplyTheme();
  return <RouterProvider router={router} />;
};

export default App;
