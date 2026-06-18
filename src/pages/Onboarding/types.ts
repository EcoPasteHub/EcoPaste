import type { FC } from "react";

export type OnboardingStepId =
  | "welcome"
  | "permissions"
  | "shortcuts"
  | "ignoreApps"
  | "legacyImport"
  | "done";

export interface OnboardingStepProps {
  onActionLoadingChange?: (loading: boolean) => void;
  onActionsChange?: (actions: OnboardingStepActions | null) => void;
  requestNext?: () => void | Promise<void>;
}

export interface OnboardingStepDefinition {
  id: OnboardingStepId;
  icon: string;
  component: FC<OnboardingStepProps>;
}

export interface OnboardingStepActions {
  extraText?: string;
  nextDisabled?: boolean;
  nextText?: string;
  onExtra?: () => void | Promise<void>;
  onNext?: () => void | Promise<void>;
}
