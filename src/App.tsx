import type { Event, UnlistenFn } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { useEventListener, useMount, useUnmount } from "ahooks";
import { App as AntdApp, ConfigProvider, theme } from "antd";
import type { FC } from "react";
import { use, useEffect, useRef, useState } from "react";
import { RouterProvider } from "react-router";
import { useSnapshot } from "valtio";
import { router } from "./router";
import { settingsReady, settingsState } from "./stores/settings";
import type { Theme as SettingsTheme } from "./types/settings";
import { log } from "./utils/log";

type ResolvedTheme = "light" | "dark";

/**
 * 根据用户设置与系统偏好解析当前实际主题。
 */
const resolveTheme = (mode: SettingsTheme, systemTheme: ResolvedTheme) => {
  if (mode === "dark") return "dark";
  if (mode === "light") return "light";

  return systemTheme;
};

/**
 * 把 Tauri 返回的窗口主题归一到前端可用的 light / dark。
 */
const normalizeTauriTheme = (value: ResolvedTheme | null): ResolvedTheme => {
  if (value === "dark") return "dark";

  return "light";
};

/**
 * 等待 Rust 设置首屏快照灌入后再渲染，避免组件读到空对象闪烁默认值。
 * `use()` 在 promise pending 时抛出，由父级（`main.tsx`）的 Suspense 接住。
 */
const App: FC = () => {
  use(settingsReady);

  const settings = useSnapshot(settingsState);
  const themeUnlistenRef = useRef<UnlistenFn | null>(null);
  const themeMountedRef = useRef(false);
  const [systemTheme, setSystemTheme] = useState<ResolvedTheme>("light");
  const mode = settings.appearance.theme;
  const resolvedTheme = resolveTheme(mode, systemTheme);
  const algorithm =
    resolvedTheme === "dark" ? theme.darkAlgorithm : theme.defaultAlgorithm;

  /**
   * 接收 Tauri 系统主题变化事件，驱动 `auto` 模式的实际主题。
   */
  const handleTauriThemeChanged = (event: Event<ResolvedTheme>) => {
    setSystemTheme(event.payload);
  };

  /**
   * 初始化 Tauri 主题快照与系统主题变化监听。
   */
  const initializeTauriThemeListener = async () => {
    try {
      const currentWindow = getCurrentWebviewWindow();
      const currentTheme = await currentWindow.theme();
      const unlisten = await currentWindow.onThemeChanged(
        handleTauriThemeChanged,
      );

      setSystemTheme(normalizeTauriTheme(currentTheme));

      if (!themeMountedRef.current) {
        unlisten();
        return;
      }

      themeUnlistenRef.current = unlisten;
    } catch (error) {
      log.error("tauri theme listener failed", error);
    }
  };

  /**
   * 移除 Tauri 系统主题变化监听。
   */
  const cleanupTauriThemeListener = () => {
    themeMountedRef.current = false;

    if (!themeUnlistenRef.current) return;

    themeUnlistenRef.current();
    themeUnlistenRef.current = null;
  };

  useMount(() => {
    themeMountedRef.current = true;
    void initializeTauriThemeListener();
  });

  useUnmount(cleanupTauriThemeListener);

  useEffect(() => {
    const root = document.documentElement;
    root.classList.toggle("dark", resolvedTheme === "dark");
    root.classList.toggle("light", resolvedTheme === "light");
  }, [resolvedTheme]);

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
    <ConfigProvider theme={{ algorithm, cssVar: { key: "eco-paste" } }}>
      <AntdApp>
        <RouterProvider router={router} />
      </AntdApp>
    </ConfigProvider>
  );
};

export default App;
