import { Card } from "antd";
import type { FC, ReactNode } from "react";
import { cn } from "@/utils/cn";

interface OnboardingCardProps {
  action?: ReactNode;
  children?: ReactNode;
  className?: string;
  description: ReactNode;
  icon: string;
  title: ReactNode;
}

const OnboardingCard: FC<OnboardingCardProps> = (props) => {
  const { action, children, description, icon, title } = props;
  const control = action ?? children;

  return (
    <Card>
      <div className="flex h-full">
        <div className="min-w-0 flex-1">
          <div className="mb-4 flex min-w-0 items-center gap-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-2 bg-ant-fill-quaternary text-ant-primary">
              <i aria-hidden="true" className={cn(icon, "text-xl")} />
            </span>
            <h2 className="m-0 font-semibold text-ant-text text-lg">{title}</h2>
          </div>

          <p className="m-0 text-ant-secondary text-sm leading-relaxed">
            {description}
          </p>
        </div>

        {control ? <div className="ml-8 shrink-0">{control}</div> : null}
      </div>
    </Card>
  );
};

export default OnboardingCard;
