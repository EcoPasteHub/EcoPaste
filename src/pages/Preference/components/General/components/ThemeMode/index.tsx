import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { useMount } from "ahooks";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import ProSelect from "@/components/ProSelect";
import { useImmediateKey } from "@/hooks/useImmediateKey";
import { globalStore } from "@/stores/global";
import type { Theme } from "@/types/store";

interface Option {
  label: string;
  value: Theme;
}

const appWindow = getCurrentWebviewWindow();

const ThemeMode = () => {
  const { appearance } = useSnapshot(globalStore);
  const { t } = useTranslation();

  useMount(() => {
    // 监听系统主题的变化
    appWindow.onThemeChanged(async ({ payload }) => {
      if (globalStore.appearance.theme !== "auto") return;

      globalStore.appearance.isDark = payload === "dark";
    });
  });

  useImmediateKey(globalStore.appearance, "theme", async (value) => {
    let nextTheme = value === "auto" ? null : value;

    await appWindow.setTheme(nextTheme);

    nextTheme = nextTheme ?? (await appWindow.theme());

    globalStore.appearance.isDark = nextTheme === "dark";
  });

  const options: Option[] = [
    {
      label: t("preference.settings.appearance_settings.label.theme_auto"),
      value: "auto",
    },
    {
      label: t("preference.settings.appearance_settings.label.theme_light"),
      value: "light",
    },
    {
      label: t("preference.settings.appearance_settings.label.theme_dark"),
      value: "dark",
    },
  ];

  return (
    <ProSelect
      onChange={(value) => {
        globalStore.appearance.theme = value;
      }}
      options={options}
      title={t("preference.settings.appearance_settings.label.theme")}
      value={appearance.theme}
    />
  );
};

export default ThemeMode;
