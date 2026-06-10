import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { useEventListener, useMount } from "ahooks";
import type { ConfigProviderProps } from "antd";
import { App as AntdApp, ConfigProvider } from "antd";
import enUS from "antd/locale/en_US";
import zhCN from "antd/locale/zh_CN";
import type { FC } from "react";
import { use, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { RouterProvider } from "react-router";
import { useSnapshot } from "valtio";
import { notifyWindowReady } from "@/commands";
import { useAppTheme } from "@/hooks/useAppTheme";
import { router } from "./router";
import { settingsReady, settingsState } from "./stores/settings";
import "./stores/windowLifecycle";
import type { Language } from "./types/settings";
import { log } from "./utils/log";

const ANTD_MODAL_CONFIG = {
  centered: true,
} satisfies ConfigProviderProps["modal"];

/**
 * 把设置语言映射到 Ant Design 内置 locale。
 */
const resolveAntdLocale = (language: Language) => {
  if (language === "en-US") return enUS;

  return zhCN;
};

/**
 * 等待 Rust 设置首屏快照灌入后再渲染，避免组件读到空对象闪烁默认值。
 * `use()` 在 promise pending 时抛出，由父级（`main.tsx`）的 Suspense 接住。
 */
const App: FC = () => {
  use(settingsReady);

  const { i18n } = useTranslation();
  const settings = useSnapshot(settingsState);
  const mode = settings.appearance.theme;
  const language = settings.appearance.language;
  const antdTheme = useAppTheme(mode);
  const locale = resolveAntdLocale(language);

  useEffect(() => {
    document.documentElement.lang = language;

    if (i18n.language === language) return;

    void i18n.changeLanguage(language);
  }, [i18n, language]);

  // settingsReady 已由 use() gate，挂载即视为前端基础初始化完成；回报 Rust 推进窗口到 ready 阶段。
  // notifyWindowReady 内部已吞掉并记录失败，这里无需再 try/catch。
  useMount(async () => {
    await notifyWindowReady(getCurrentWebviewWindow().label);
  });

  // 兜底未捕获的 Promise rejection：统一进日志通道，避免只在 devtools 红字闪过、生产环境完全无痕。
  useEventListener("unhandledrejection", (event) => {
    const { reason } = event;

    log.error(
      "unhandled promise rejection",
      reason instanceof Error ? reason : { reason },
    );
  });

  // 兜底未捕获的同步异常（含资源加载错误）。React 渲染错误由 ErrorBoundary 接，不会走到这里。
  useEventListener("error", (event) => {
    const { error, ...rest } = event;

    log.error("uncaught error", error instanceof Error ? error : rest);
  });

  return (
    <ConfigProvider locale={locale} modal={ANTD_MODAL_CONFIG} theme={antdTheme}>
      <AntdApp>
        <RouterProvider router={router} />
      </AntdApp>
    </ConfigProvider>
  );
};

export default App;
