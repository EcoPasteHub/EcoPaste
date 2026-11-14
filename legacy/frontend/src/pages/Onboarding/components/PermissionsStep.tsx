import type { FC } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import { findPreferenceSectionSettings } from "@/pages/Preference/config/preferenceSchema";
import { commitSettingChange } from "@/pages/Preference/services/preferenceSettings";
import type {
  PreferenceSetting,
  SettingValue,
} from "@/pages/Preference/types/preferences";
import { settingsState } from "@/stores/settings";
import type { Settings } from "@/types/settings";
import OnboardingPreferenceCard from "./OnboardingPreferenceCard";
import OnboardingStepLayout from "./OnboardingStepLayout";

const PERMISSION_SETTINGS = findPreferenceSectionSettings("permissions");
const DESCRIPTION_PLATFORM = PERMISSION_SETTINGS.some((setting) => {
  return setting.id === "permissions.runAsAdministrator";
})
  ? "windows"
  : "macos";

const PermissionsStep: FC = () => {
  const { t } = useTranslation("onboarding");
  const settings = useSnapshot(settingsState) as Settings;

  const handleChange = async (
    setting: PreferenceSetting,
    value: SettingValue,
  ) => {
    await commitSettingChange(setting, value);
  };

  return (
    <OnboardingStepLayout
      contentClassName="flex flex-col gap-4"
      description={t(`permissions.description.${DESCRIPTION_PLATFORM}`)}
      icon={<i aria-hidden="true" className="i-lucide:shield-check" />}
      title={t("permissions.title")}
    >
      {PERMISSION_SETTINGS.map((setting) => {
        return (
          <OnboardingPreferenceCard
            key={setting.id}
            onChange={handleChange}
            setting={setting}
            settings={settings}
          />
        );
      })}
    </OnboardingStepLayout>
  );
};

export default PermissionsStep;
