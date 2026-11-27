import type { FC, ReactNode } from "react";
import { cn } from "@/utils/cn";

interface StepHeroProps {
  description: ReactNode;
  icon: ReactNode;
  iconClassName?: string;
  title: ReactNode;
}

const StepHero: FC<StepHeroProps> = (props) => {
  const { description, icon, iconClassName, title } = props;

  return (
    <header className="flex shrink-0 flex-col items-center text-center">
      <div
        className={cn(
          "mb-3 flex size-12 items-center justify-center overflow-hidden text-4xl text-ant-primary leading-none",
          iconClassName,
        )}
      >
        {icon}
      </div>
      <h1 className="m-0 font-semibold text-2xl text-ant-text leading-tight">
        {title}
      </h1>
      <p className="m-0 mt-2 text-ant-secondary text-sm leading-relaxed">
        {description}
      </p>
    </header>
  );
};

export default StepHero;
