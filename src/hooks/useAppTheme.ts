import type { Event, UnlistenFn } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { useMount, useUnmount } from "ahooks";
import { type ThemeConfig, theme } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";
import type { Theme as SettingsTheme } from "@/types/settings";
import { log } from "@/utils/log";

type ResolvedTheme = "light" | "dark";
type NativeTheme = ResolvedTheme | null;

/**
 * 根据用户设置与系统偏好解析当前实际主题。
 */
const resolveTheme = (mode: SettingsTheme, systemTheme: ResolvedTheme) => {
  if (mode === "dark") return "dark";
  if (mode === "light") return "light";

  return systemTheme;
};

/**
 * 把应用主题设置转换成 Tauri 原生窗口主题；null 表示跟随系统。
 */
const resolveNativeTheme = (mode: SettingsTheme): NativeTheme => {
  if (mode === "auto") return null;

  return mode;
};

/**
 * 把 Tauri 返回的窗口主题归一到前端可用的 light / dark。
 */
const normalizeTauriTheme = (value: ResolvedTheme | null): ResolvedTheme => {
  if (value === "dark") return "dark";

  return "light";
};

/**
 * 同步当前 webview window 的原生主题，并在 auto 模式下回读实际系统主题。
 */
const syncTauriWindowTheme = async (
  mode: SettingsTheme,
): Promise<ResolvedTheme | null> => {
  const currentWindow = getCurrentWebviewWindow();
  const nativeTheme = resolveNativeTheme(mode);

  await currentWindow.setTheme(nativeTheme);

  if (nativeTheme !== null) return null;

  return normalizeTauriTheme(await currentWindow.theme());
};

/**
 * 解析应用主题并同步系统主题变化、html class 与 Ant Design token 算法。
 */
export const useAppTheme = (mode: SettingsTheme): ThemeConfig => {
  const themeUnlistenRef = useRef<UnlistenFn | null>(null);
  const themeMountedRef = useRef(false);
  const [systemTheme, setSystemTheme] = useState<ResolvedTheme>("light");
  const resolvedTheme = resolveTheme(mode, systemTheme);
  const algorithm =
    resolvedTheme === "dark" ? theme.darkAlgorithm : theme.defaultAlgorithm;
  const antdTheme = useMemo(() => {
    return { algorithm };
  }, [algorithm]);

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

  useEffect(() => {
    let stale = false;

    /**
     * 将设置里的主题模式同步给 Tauri 原生窗口，覆盖标题栏等非 Web 区域。
     */
    const syncNativeTheme = async () => {
      try {
        const currentTheme = await syncTauriWindowTheme(mode);

        if (stale || currentTheme === null) return;

        setSystemTheme(currentTheme);
      } catch (error) {
        log.error("tauri native theme sync failed", error);
      }
    };

    void syncNativeTheme();

    return () => {
      stale = true;
    };
  }, [mode]);

  return antdTheme;
};
