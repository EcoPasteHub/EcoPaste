import { Switch } from "antd";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import type { ShortcutRecorderConflict } from "@/components/ShortcutRecorder";
import ShortcutRecorder from "@/components/ShortcutRecorder";
import { settingsState, updateSettings } from "@/stores/settings";
import type { Shortcuts } from "@/types/settings";
import { isWin } from "@/utils/is";
import OnboardingCard from "./OnboardingCard";
import OnboardingStepLayout from "./OnboardingStepLayout";

type ShortcutRecorderKey = "openClipboard" | "openPreference";

const SHORTCUT_RECORDER_KEYS: ShortcutRecorderKey[] = [
  "openClipboard",
  "openPreference",
];

const SHORTCUT_ICONS = {
  openClipboard: "i-lucide:clipboard",
  openPreference: "i-lucide:settings",
  winV: "i-lucide:clipboard-list",
} as const;

const ShortcutsStep: FC = () => {
  const { t } = useTranslation("onboarding");
  const settings = useSnapshot(settingsState);

  const handleShortcutChange = async (
    key: ShortcutRecorderKey,
    value: string,
  ) => {
    const shortcutsPatch: Partial<Shortcuts> = {
      [key]: value,
    };

    await updateSettings({
      shortcuts: shortcutsPatch,
    });
  };

  const handleWinVChange = async (checked: boolean) => {
    await updateSettings({
      shortcuts: {
        winV: checked,
      },
    });
  };

  const buildConflicts = (
    key: ShortcutRecorderKey,
  ): ShortcutRecorderConflict[] => {
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
      contentClassName={isWin ? "flex flex-col gap-3" : "flex flex-col gap-4"}
      description={t("shortcuts.description")}
      icon={<i aria-hidden="true" className="i-lucide:keyboard" />}
      title={t("shortcuts.title")}
    >
      {SHORTCUT_RECORDER_KEYS.map((key) => {
        const handleChange = async (value: string) => {
          await handleShortcutChange(key, value);
        };

        return (
          <OnboardingCard
            compact={isWin}
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

      {isWin ? (
        <OnboardingCard
          compact
          description={t("shortcuts.items.winV.description")}
          icon={SHORTCUT_ICONS.winV}
          title={t("shortcuts.items.winV.title")}
        >
          <Switch
            checked={settings.shortcuts.winV}
            onChange={handleWinVChange}
          />
        </OnboardingCard>
      ) : null}
    </OnboardingStepLayout>
  );
};

export default ShortcutsStep;
