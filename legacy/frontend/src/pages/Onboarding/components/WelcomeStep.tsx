import type { FC } from "react";
import { useTranslation } from "react-i18next";
import OnboardingCard from "./OnboardingCard";
import OnboardingStepLayout from "./OnboardingStepLayout";

const FEATURE_KEYS = ["capture", "search", "reuse"] as const;

const FEATURE_ICONS = {
  capture: "i-lucide:clipboard-plus",
  reuse: "i-lucide:zap",
  search: "i-lucide:search",
} as const;

const WelcomeStep: FC = () => {
  const { t } = useTranslation("onboarding");

  return (
    <OnboardingStepLayout
      contentClassName="grid grid-cols-3 gap-4"
      description={t("welcome.description")}
      icon={
        <img
          alt={t("welcome.logoAlt")}
          className="size-full rounded-3 object-contain"
          src="/logo.png"
        />
      }
      iconClassName="size-14"
      title={t("welcome.title")}
    >
      {FEATURE_KEYS.map((key) => {
        return (
          <OnboardingCard
            description={t(`welcome.features.${key}.description`)}
            icon={FEATURE_ICONS[key]}
            key={key}
            title={t(`welcome.features.${key}.title`)}
          />
        );
      })}
    </OnboardingStepLayout>
  );
};

export default WelcomeStep;
