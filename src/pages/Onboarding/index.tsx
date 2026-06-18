import type { FC } from "react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import { finishOnboarding, setOnboardingStep } from "@/commands";
import { settingsState } from "@/stores/settings";
import OnboardingActions from "./components/OnboardingActions";
import OnboardingShell from "./components/OnboardingShell";
import { ONBOARDING_STEPS } from "./steps";
import type { OnboardingStepActions } from "./types";

const LAST_STEP_INDEX = ONBOARDING_STEPS.length - 1;

const Onboarding: FC = () => {
  const { t } = useTranslation("onboarding");
  const settings = useSnapshot(settingsState);
  const [activeIndex, setActiveIndex] = useState(() => {
    return clampStepIndex(settingsState.onboarding.lastStep);
  });
  const [transitionDirection, setTransitionDirection] = useState<1 | -1>(1);
  const [actionLoading, setActionLoading] = useState(false);
  const [stepActions, setStepActions] = useState<OnboardingStepActions | null>(
    null,
  );
  const [finishLoading, setFinishLoading] = useState(false);

  useEffect(() => {
    setActiveIndex(clampStepIndex(settings.onboarding.lastStep));
  }, [settings.onboarding.lastStep]);

  const currentStep = ONBOARDING_STEPS[activeIndex] ?? ONBOARDING_STEPS[0];
  const StepComponent = currentStep.component;
  const isFirstStep = activeIndex === 0;
  const isLastStep = activeIndex === LAST_STEP_INDEX;
  const nextText = isFirstStep
    ? t("actions.start")
    : isLastStep
      ? t("actions.finish")
      : t("actions.next");

  const moveToStep = async (nextIndex: number) => {
    const normalizedIndex = clampStepIndex(nextIndex);
    const direction = normalizedIndex >= activeIndex ? 1 : -1;

    setTransitionDirection(direction);

    await setOnboardingStep(normalizedIndex);
    setActiveIndex(normalizedIndex);
  };

  const handlePrevious = async () => {
    await moveToStep(activeIndex - 1);
  };

  const handleNextStep = async () => {
    await moveToStep(activeIndex + 1);
  };

  const handleFinish = async () => {
    setFinishLoading(true);

    try {
      await finishOnboarding();
    } finally {
      setFinishLoading(false);
    }
  };

  const handleNext = async () => {
    if (isLastStep) {
      await handleFinish();
      return;
    }

    await handleNextStep();
  };

  const handleStepExtra = async () => {
    await stepActions?.onExtra?.();
  };

  const handleStepNext = async () => {
    if (stepActions?.onNext) {
      await stepActions.onNext();
      return;
    }

    await handleNext();
  };

  return (
    <OnboardingShell
      actions={
        <OnboardingActions
          backText={t("actions.previous")}
          extraText={stepActions?.extraText}
          loading={actionLoading || finishLoading}
          nextDisabled={stepActions?.nextDisabled ?? false}
          nextText={stepActions?.nextText ?? nextText}
          onBack={handlePrevious}
          onExtra={stepActions?.onExtra ? handleStepExtra : void 0}
          onNext={handleStepNext}
          showBack={!isFirstStep}
        />
      }
      activeIndex={activeIndex}
      activeStepId={currentStep.id}
      direction={transitionDirection}
      steps={ONBOARDING_STEPS}
    >
      <StepComponent
        onActionLoadingChange={setActionLoading}
        onActionsChange={setStepActions}
        requestNext={handleNextStep}
      />
    </OnboardingShell>
  );
};

export default Onboarding;

function clampStepIndex(step: number) {
  return Math.max(0, Math.min(step, LAST_STEP_INDEX));
}
