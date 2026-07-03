import type { FC } from "react";
import { useTranslation } from "react-i18next";
import PermissionControl from "@/pages/Preference/components/settingControls/PermissionControl";
import { allPreferenceSettings } from "@/pages/Preference/config/preferenceSchema";
import { isMac } from "@/utils/is";
import type { OnboardingStepProps } from "../types";
import OnboardingStepLayout from "./OnboardingStepLayout";
import PreferenceSettingCard from "./PreferenceSettingCard";

const permissionSettings = allPreferenceSettings
  .filter(({ setting }) => {
    return setting.control.type === "permission";
  })
  .map(({ setting }) => {
    return setting;
  });

/**
 * 引导页权限步骤复用偏好设置里的权限控件和文案，避免授权弹窗与设置说明分叉。
 */
const PermissionsStep: FC<OnboardingStepProps> = () => {
  const { t } = useTranslation("onboarding");

  return (
    <OnboardingStepLayout
      contentClassName="flex flex-col gap-4"
      description={t(`permissions.description.${isMac ? "macos" : "windows"}`)}
      icon={<i aria-hidden="true" className="i-lucide:shield-check" />}
      title={t("permissions.title")}
    >
      {permissionSettings.map((setting) => {
        return (
          <PreferenceSettingCard
            action={<PermissionControl setting={setting} />}
            key={setting.id}
            setting={setting}
          />
        );
      })}
    </OnboardingStepLayout>
  );
};

export default PermissionsStep;
