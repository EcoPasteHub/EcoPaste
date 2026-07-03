import type { FC, ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { resolveSettingVisual } from "@/pages/Preference/components/settingControls/settingVisual";
import type { PreferenceSetting } from "@/pages/Preference/types/preferences";
import { translatePreferenceSetting } from "@/pages/Preference/utils/preferenceI18n";
import OnboardingCard from "./OnboardingCard";

interface PreferenceSettingCardProps {
  action?: ReactNode;
  children?: ReactNode;
  compact?: boolean;
  setting: PreferenceSetting;
}

/**
 * 用偏好设置 schema 的文案和视觉定义渲染引导页卡片。
 */
const PreferenceSettingCard: FC<PreferenceSettingCardProps> = (props) => {
  const { action, children, compact = false, setting } = props;
  const { t } = useTranslation("preferences");
  const visual = resolveSettingVisual(setting.id);

  return (
    <OnboardingCard
      action={action}
      compact={compact}
      description={translatePreferenceSetting(t, setting, "description")}
      icon={visual.icon}
      title={translatePreferenceSetting(t, setting, "title")}
    >
      {children}
    </OnboardingCard>
  );
};

export default PreferenceSettingCard;
