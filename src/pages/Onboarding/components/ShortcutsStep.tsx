import type { FC } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import type { ShortcutRecorderConflict } from "@/components/ShortcutRecorder";
import ShortcutRecorder from "@/components/ShortcutRecorder";
import { settingsState, updateSettings } from "@/stores/settings";
import type { Shortcuts } from "@/types/settings";
import OnboardingCard from "./OnboardingCard";
import OnboardingStepLayout from "./OnboardingStepLayout";

type ShortcutKey = "openClipboard" | "openPreference";

const SHORTCUT_KEYS: ShortcutKey[] = ["openClipboard", "openPreference"];

const SHORTCUT_ICONS = {
  openClipboard: "i-lucide:clipboard",
  openPreference: "i-lucide:settings",
} as const;

const ShortcutsStep: FC = () => {
  const { t } = useTranslation("onboarding");
  const settings = useSnapshot(settingsState);

  const handleShortcutChange = async (key: ShortcutKey, value: string) => {
    const shortcutsPatch: Partial<Shortcuts> = {
      [key]: value,
    };

    await updateSettings({
      shortcuts: shortcutsPatch,
    });
  };

  const buildConflicts = (key: ShortcutKey): ShortcutRecorderConflict[] => {
    if (key === "openClipboard") {
      return [
        {
          label: t("shortcuts.items.openPreference.title"),
          value: settings.shortcuts.openPreference,
        },
      ];
    }

    if (key === "openPreference") {
      return [
        {
          label: t("shortcuts.items.openClipboard.title"),
          value: settings.shortcuts.openClipboard,
        },
      ];
    }

    return [];
  };

  return (
    <OnboardingStepLayout
      contentClassName="flex flex-col gap-4"
      description={t("shortcuts.description")}
      icon={<i aria-hidden="true" className="i-lucide:keyboard" />}
      title={t("shortcuts.title")}
    >
      {SHORTCUT_KEYS.map((key) => {
        const handleChange = async (value: string) => {
          await handleShortcutChange(key, value);
        };

        return (
          <OnboardingCard
            description={t(`shortcuts.items.${key}.description`)}
            icon={SHORTCUT_ICONS[key]}
            key={key}
            title={t(`shortcuts.items.${key}.title`)}
          >
            <ShortcutRecorder
              conflicts={buildConflicts(key)}
              onChange={handleChange}
              value={settings.shortcuts[key]}
            />
          </OnboardingCard>
        );
      })}
    </OnboardingStepLayout>
  );
};

export default ShortcutsStep;
