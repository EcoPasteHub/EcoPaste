import type { FC } from "react";
import { useTranslation } from "react-i18next";
import OnboardingCard from "./OnboardingCard";
import OnboardingStepLayout from "./OnboardingStepLayout";

const CARD_KEYS = ["open", "search", "preferences"] as const;

const CARD_ICONS = {
  open: "i-lucide:panel-top-open",
  preferences: "i-lucide:sliders-horizontal",
  search: "i-lucide:search",
} as const;

const DoneStep: FC = () => {
  const { t } = useTranslation("onboarding");

  return (
    <OnboardingStepLayout
      contentClassName="grid grid-cols-3 gap-4"
      description={t("done.description")}
      icon={
        <img
          alt={t("done.logoAlt")}
          className="size-full rounded-3 object-contain"
          src="/logo.png"
        />
      }
      iconClassName="size-14"
      title={t("done.title")}
    >
      {CARD_KEYS.map((key) => {
        return (
          <OnboardingCard
            description={t(`done.cards.${key}.description`)}
            icon={CARD_ICONS[key]}
            key={key}
            title={t(`done.cards.${key}.title`)}
          />
        );
      })}
    </OnboardingStepLayout>
  );
};

export default DoneStep;
