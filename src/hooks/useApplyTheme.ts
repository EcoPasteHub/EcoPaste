import { getCurrentWindow } from "@tauri-apps/api/window";
import { useEffect } from "react";
import { useSnapshot } from "valtio";
import { settingsState } from "@/stores/settings";
import { log } from "@/utils/log";

/**
 * 主题应用要两条路径都走：
 * 1) Tauri 窗口侧 setTheme（null=跟随系统）—— 让原生 chrome（标题栏装饰、滚动条、菜单等）跟随；
 *    仅改 DOM class 不改窗口会导致原生区域与内容主题割裂。
 * 2) html 上的 light/dark class —— 供 UnoCSS `dark:` 变体识别（antd 主题算法由 App.tsx 内
 *    useResolvedMode 单独驱动；这里只负责 DOM class 和窗口 chrome）。
 * auto 模式下用窗口的 onThemeChanged 而不是 matchMedia——窗口事件已经由 OS 触发，且 setTheme(null) 后 theme() 直接给到解析结果，单一信源避免双订阅打架。
 */
export function useApplyTheme(): void {
  const { value, loaded } = useSnapshot(settingsState);
  const theme = value?.appearance.theme;

  useEffect(() => {
    if (!loaded || !theme) {
      return;
    }

    const appWindow = getCurrentWindow();
    let unlisten: (() => void) | null = null;
    let cancelled = false;

    const applyClass = (mode: "light" | "dark") => {
      const root = document.documentElement;
      root.classList.toggle("dark", mode === "dark");
      root.classList.toggle("light", mode === "light");
      root.dataset.theme = mode;
      root.style.colorScheme = mode;
    };

    (async () => {
      // null = 跟随系统；显式 light/dark 则锁死窗口主题。
      await appWindow.setTheme(theme === "auto" ? null : theme);
      if (cancelled) return;

      const resolved = (await appWindow.theme()) ?? "light";
      if (cancelled) return;
      applyClass(resolved);

      if (theme === "auto") {
        unlisten = await appWindow.onThemeChanged(({ payload }) => {
          applyClass(payload);
        });
        if (cancelled) unlisten();
      }
    })().catch((err) => {
      log.error("apply theme failed", err);
    });

    return () => {
      cancelled = true;
      unlisten?.();
    };
  }, [theme, loaded]);
}
