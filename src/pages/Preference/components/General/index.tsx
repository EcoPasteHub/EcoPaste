import { disable, enable, isEnabled } from "@tauri-apps/plugin-autostart";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import ProList from "@/components/ProList";
import ProSwitch from "@/components/ProSwitch";
import { useImmediateKey } from "@/hooks/useImmediateKey";
import { globalStore } from "@/stores/global";
import { isMac } from "@/utils/is";
import Language from "./components/Language";
import MacosPermissions from "./components/MacosPermissions";
import ThemeMode from "./components/ThemeMode";

const General = () => {
  const { app, update } = useSnapshot(globalStore);
  const { t } = useTranslation();

  // 监听自动启动变更
  useImmediateKey(globalStore.app, "autoStart", async (value) => {
    const enabled = await isEnabled();

    if (value && !enabled) {
      return enable();
    }

    if (!value && enabled) {
      disable();
    }
  });

  return (
    <>
      {isMac && <MacosPermissions />}

      <ProList header={t("preference.settings.app_settings.title")}>
        <ProSwitch
          onChange={(value) => {
            globalStore.app.autoStart = value;
          }}
          title={t("preference.settings.app_settings.label.auto_start")}
          value={app.autoStart}
        />

        <ProSwitch
          description={t("preference.settings.app_settings.hints.silent_start")}
          onChange={(value) => {
            globalStore.app.silentStart = value;
          }}
          title={t("preference.settings.app_settings.label.silent_start")}
          value={app.silentStart}
        />

        <ProSwitch
          onChange={(value) => {
            globalStore.app.showMenubarIcon = value;
          }}
          title={t("preference.settings.app_settings.label.show_menubar_icon")}
          value={app.showMenubarIcon}
        />

        <ProSwitch
          onChange={(value) => {
            globalStore.app.showTaskbarIcon = value;
          }}
          title={t("preference.settings.app_settings.label.show_taskbar_icon")}
          value={app.showTaskbarIcon}
        />
      </ProList>

      <ProList header={t("preference.settings.appearance_settings.title")}>
        <Language />

        <ThemeMode />
      </ProList>

      <ProList header={t("preference.settings.update_settings.title")}>
        <ProSwitch
          onChange={(value) => {
            globalStore.update.auto = value;
          }}
          title={t("preference.settings.update_settings.label.auto_update")}
          value={update.auto}
        />

        <ProSwitch
          description={t(
            "preference.settings.update_settings.hints.update_beta",
          )}
          onChange={(value) => {
            globalStore.update.beta = value;
          }}
          title={t("preference.settings.update_settings.label.update_beta")}
          value={update.beta}
        />
      </ProList>
    </>
  );
};

export default General;
