import type { FC } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/utils/cn";
import type { PreferenceSetting } from "../types/preferences";

interface PreferenceStatusBadgeProps {
  compact?: boolean;
  status: PreferenceSetting["status"];
}

const STATUS_META: Record<
  NonNullable<PreferenceSetting["status"]>,
  { icon: string; labelKey: string }
> = {
  alwaysOn: { icon: "i-lucide:check", labelKey: "status.alwaysOn" },
  comingSoon: { icon: "i-lucide:sparkles", labelKey: "status.comingSoon" },
  experimental: {
    icon: "i-lucide:flask-conical",
    labelKey: "status.experimental",
  },
  requiresBackend: {
    icon: "i-lucide:plug",
    labelKey: "status.requiresBackend",
  },
};

/**
 * 设置项状态徽标：用文字和图标表达状态，避免只依赖颜色。
 */
const PreferenceStatusBadge: FC<PreferenceStatusBadgeProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { compact = false, status } = props;

  if (!status) return null;
  if (compact && status === "alwaysOn") return null;

  const meta = STATUS_META[status];

  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center gap-1 whitespace-nowrap rounded-full border border-ant-border-secondary bg-ant-fill-quaternary text-ant-secondary leading-none",
        compact ? "px-1.5 py-0.75 text-xs" : "px-1.75 py-1 text-xs",
      )}
    >
      <i aria-hidden="true" className={meta.icon} />
      {t(meta.labelKey)}
    </span>
  );
};

export default PreferenceStatusBadge;
