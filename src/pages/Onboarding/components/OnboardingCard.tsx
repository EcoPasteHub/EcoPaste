import { Card } from "antd";
import type { FC, ReactNode } from "react";
import { cn } from "@/utils/cn";

interface OnboardingCardProps {
  action?: ReactNode;
  children?: ReactNode;
  className?: string;
  compact?: boolean;
  description: ReactNode;
  icon: string;
  title: ReactNode;
}

const OnboardingCard: FC<OnboardingCardProps> = (props) => {
  const {
    action,
    children,
    className,
    compact = false,
    description,
    icon,
    title,
  } = props;
  const control = action ?? children;

  return (
    <Card className={className} classNames={{ body: cn({ "!p-4": compact }) }}>
      <div className="flex h-full">
        <div className="min-w-0 flex-1">
          <div
            className={cn("flex min-w-0 items-center gap-3", {
              "mb-2": compact,
              "mb-4": !compact,
            })}
          >
            <span
              className={cn(
                "flex shrink-0 items-center justify-center rounded-2 bg-ant-fill-quaternary text-ant-primary",
                {
                  "size-8": compact,
                  "size-9": !compact,
                },
              )}
            >
              <i
                aria-hidden="true"
                className={cn(icon, {
                  "text-lg": compact,
                  "text-xl": !compact,
                })}
              />
            </span>
            <h2
              className={cn("m-0 font-semibold text-ant-text", {
                "text-base": compact,
                "text-lg": !compact,
              })}
            >
              {title}
            </h2>
          </div>

          <p className="m-0 text-ant-secondary text-sm leading-relaxed">
            {description}
          </p>
        </div>

        {control ? (
          <div
            className={cn("flex shrink-0 items-center", {
              "ml-6": compact,
              "ml-8": !compact,
            })}
          >
            {control}
          </div>
        ) : null}
      </div>
    </Card>
  );
};

export default OnboardingCard;
