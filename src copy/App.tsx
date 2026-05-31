// antd v6 通过 ConfigProvider.theme.algorithm 切换主题；useApplyTheme 同步给
// <html> 挂 light/dark class 供 UnoCSS `dark:` 变体使用。

import { App as AntdApp, theme as antdTheme, ConfigProvider } from "antd";
import enUS from "antd/locale/en_US";
import zhCN from "antd/locale/zh_CN";
import { useEffect, useState } from "react";
import { RouterProvider } from "react-router";
import { useSnapshot } from "valtio";
import { TAURI_EVENT } from "@/constants/events";
import { useApplyLanguage } from "@/hooks/useApplyLanguage";
import { useApplyTheme } from "@/hooks/useApplyTheme";
import { useTauriListen } from "@/hooks/useTauriListen";
import { router } from "@/router";
import { applySettings, settingsState } from "@/stores/settings";
import type { Settings } from "@/types/settings";

/**
 * 把 settings.appearance.theme（user 偏好：auto/light/dark）解算成当前实际生效模式，
 * 喂给 antd ConfigProvider 的 algorithm。auto 模式下监听 prefers-color-scheme。
 */
const useResolvedMode = (): "light" | "dark" => {
  const { value } = useSnapshot(settingsState);
  const pref = value?.appearance.theme ?? "auto";
  const [systemDark, setSystemDark] = useState(
    () =>
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-color-scheme: dark)").matches,
  );

  useEffect(() => {
    if (pref !== "auto") return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = (e: MediaQueryListEvent) => setSystemDark(e.matches);
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, [pref]);

  if (pref === "dark") return "dark";
  if (pref === "light") return "light";
  return systemDark ? "dark" : "light";
};

const App = () => {
  useApplyTheme();
  useApplyLanguage();
  useTauriListen<Settings>(TAURI_EVENT.SETTINGS_UPDATED, applySettings);
  const mode = useResolvedMode();
  const { value } = useSnapshot(settingsState);
  const language = value?.appearance.language ?? "zh-CN";
  const locale = language === "en-US" ? enUS : zhCN;

  return (
    <ConfigProvider
      locale={locale}
      theme={{
        algorithm:
          mode === "dark"
            ? antdTheme.darkAlgorithm
            : antdTheme.defaultAlgorithm,
        hashed: false,
      }}
    >
      <AntdApp>
        <RouterProvider router={router} />
      </AntdApp>
    </ConfigProvider>
  );
};

export default App;
