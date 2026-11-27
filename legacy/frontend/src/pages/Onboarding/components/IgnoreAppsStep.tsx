import { useMount } from "ahooks";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import SourceAppsTransfer from "@/pages/Preference/components/SourceAppsTransfer";
import { requirePreferenceSetting } from "@/pages/Preference/config/preferenceSchema";
import { commitSettingChange } from "@/pages/Preference/services/preferenceSettings";
import type {
  PreferenceSetting,
  SettingValue,
} from "@/pages/Preference/types/preferences";
import { settingsState } from "@/stores/settings";
import { preloadSourceApps } from "@/stores/sourceApps";
import type { Settings } from "@/types/settings";
import OnboardingStepLayout from "./OnboardingStepLayout";

const EXCLUDED_APPS_SETTING = requirePreferenceSetting("source.excludedApps");

const IgnoreAppsStep: FC = () => {
  const { t } = useTranslation("onboarding");
  const settings = useSnapshot(settingsState) as Settings;

  useMount(() => {
    void preloadSourceApps();
  });

  const handleExcludedAppsChange = async (
    setting: PreferenceSetting,
    value: SettingValue,
  ) => {
    await commitSettingChange(setting, value);
  };

  return (
    <OnboardingStepLayout
      contentClassName="flex-1"
      description={t("ignoreApps.description")}
      icon={<i aria-hidden="true" className="i-lucide:ban" />}
      title={t("ignoreApps.title")}
    >
      <SourceAppsTransfer
        excludedAppsSetting={EXCLUDED_APPS_SETTING}
        onChange={handleExcludedAppsChange}
        settings={settings}
      />
    </OnboardingStepLayout>
  );
};

export default IgnoreAppsStep;
