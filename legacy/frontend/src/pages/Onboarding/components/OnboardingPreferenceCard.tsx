import type { FC } from "react";
import { useTranslation } from "react-i18next";
import PreferenceSettingControl from "@/pages/Preference/components/settingControls/PreferenceSettingControl";
import { resolveSettingVisual } from "@/pages/Preference/components/settingControls/settingVisual";
import type {
  PreferenceSetting,
  PreferenceSettingChangeHandler,
} from "@/pages/Preference/types/preferences";
import { translatePreferenceSetting } from "@/pages/Preference/utils/preferenceI18n";
import type { Settings } from "@/types/settings";
import OnboardingCard from "./OnboardingCard";

interface OnboardingPreferenceCardProps {
  compact?: boolean;
  setting: PreferenceSetting;
  settings: Settings;
  onChange: PreferenceSettingChangeHandler;
}

/**
 * 在引导页卡片样式内复用偏好设置项的文案、图标和控件行为。
 */
const OnboardingPreferenceCard: FC<OnboardingPreferenceCardProps> = (props) => {
  const { compact = false, setting, settings, onChange } = props;
  const { t } = useTranslation("preferences");
  const value = setting.value?.(settings);
  const disabled =
    setting.disabled === true || setting.disabledWhen?.(settings) === true;
  const visual = resolveSettingVisual(setting.id);

  return (
    <OnboardingCard
      compact={compact}
      description={translatePreferenceSetting(t, setting, "description")}
      icon={visual.icon}
      title={translatePreferenceSetting(t, setting, "title")}
    >
      <PreferenceSettingControl
        disabled={disabled}
        onChange={onChange}
        setting={setting}
        settings={settings}
        value={value}
      />
    </OnboardingCard>
  );
};

export default OnboardingPreferenceCard;
