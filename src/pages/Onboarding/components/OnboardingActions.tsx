import { Button } from "antd";
import type { FC, ReactNode } from "react";
import { useState } from "react";

interface OnboardingActionsProps {
  nextText: ReactNode;
  backText?: ReactNode;
  extra?: ReactNode;
  extraText?: ReactNode;
  loading?: boolean;
  nextDisabled?: boolean;
  showBack?: boolean;
  onBack?: () => void | Promise<void>;
  onExtra?: () => void | Promise<void>;
  onNext: () => void | Promise<void>;
}

const OnboardingActions: FC<OnboardingActionsProps> = (props) => {
  const {
    backText,
    extra,
    extraText,
    loading = false,
    nextDisabled = false,
    nextText,
    showBack = true,
    onBack,
    onExtra,
    onNext,
  } = props;
  const [pending, setPending] = useState(false);
  const busy = loading || pending;

  const handleBackClick = async () => {
    if (busy || !onBack) return;

    setPending(true);

    try {
      await onBack();
    } finally {
      setPending(false);
    }
  };

  const handleNextClick = async () => {
    if (busy || nextDisabled) return;

    setPending(true);

    try {
      await onNext();
    } finally {
      setPending(false);
    }
  };

  const handleExtraClick = async () => {
    if (busy || !onExtra) return;

    setPending(true);

    try {
      await onExtra();
    } finally {
      setPending(false);
    }
  };

  return (
    <footer className="absolute inset-x-0 bottom-0 z-10 flex h-18 items-center justify-between px-10">
      <div>
        {showBack && (
          <Button disabled={busy} onClick={handleBackClick} size="large">
            {backText}
          </Button>
        )}
      </div>

      <div className="flex items-center gap-3">
        {extra}
        {onExtra && (
          <Button disabled={busy} onClick={handleExtraClick} size="large">
            {extraText}
          </Button>
        )}
        <Button
          disabled={nextDisabled}
          loading={busy}
          onClick={handleNextClick}
          size="large"
          type="primary"
        >
          {nextText}
        </Button>
      </div>
    </footer>
  );
};

export default OnboardingActions;
