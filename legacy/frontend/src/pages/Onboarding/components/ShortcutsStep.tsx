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

const SHORTCUT_SETTINGS = findPreferenceSectionSettings("globalShortcuts");

const ShortcutsStep: FC = () => {
  const { t } = useTranslation("onboarding");
  const settings = useSnapshot(settingsState) as Settings;
  const compact = SHORTCUT_SETTINGS.length > 2;

  const handleChange = async (
    setting: PreferenceSetting,
    value: SettingValue,
  ) => {
    await commitSettingChange(setting, value);
  };

  return (
    <OnboardingStepLayout
      contentClassName={compact ? "flex flex-col gap-3" : "flex flex-col gap-4"}
      description={t("shortcuts.description")}
      icon={<i aria-hidden="true" className="i-lucide:keyboard" />}
      title={t("shortcuts.title")}
    >
      {SHORTCUT_SETTINGS.map((setting) => {
        return (
          <OnboardingPreferenceCard
            compact={compact}
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

export default ShortcutsStep;
