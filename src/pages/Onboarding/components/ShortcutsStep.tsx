import type { FC } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import type { ShortcutRecorderConflict } from "@/components/ShortcutRecorder";
import ShortcutRecorder from "@/components/ShortcutRecorder";
import SwitchControl from "@/pages/Preference/components/settingControls/SwitchControl";
import {
  findPreferenceSetting,
  resolvePreferenceSetting,
} from "@/pages/Preference/config/preferenceSchema";
import type {
  PreferenceSetting,
  SettingValue,
} from "@/pages/Preference/types/preferences";
import {
  translatePreferencePlaceholder,
  translatePreferenceSetting,
} from "@/pages/Preference/utils/preferenceI18n";
import { settingsState, updateSettings } from "@/stores/settings";
import type { Shortcuts } from "@/types/settings";
import { isWin } from "@/utils/is";
import OnboardingStepLayout from "./OnboardingStepLayout";
import PreferenceSettingCard from "./PreferenceSettingCard";

type ShortcutRecorderKey = "openClipboard" | "openPreference";

const SHORTCUT_RECORDER_KEYS: ShortcutRecorderKey[] = [
  "openClipboard",
  "openPreference",
];

const SHORTCUT_SETTING_IDS = {
  openClipboard: "shortcuts.openClipboard",
  openPreference: "shortcuts.openPreference",
} as const;

const SHORTCUT_RECORDER_SETTINGS = {
  openClipboard: resolvePreferenceSetting(SHORTCUT_SETTING_IDS.openClipboard),
  openPreference: resolvePreferenceSetting(SHORTCUT_SETTING_IDS.openPreference),
};
const WIN_V_SETTING = findPreferenceSetting("shortcuts.winV");

const ShortcutsStep: FC = () => {
  const { t } = useTranslation("onboarding");
  const { t: preferenceT } = useTranslation("preferences");
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

  const handlePreferenceSettingChange = async (
    setting: PreferenceSetting,
    value: SettingValue,
  ) => {
    if (setting.id !== "shortcuts.winV") return;

    await updateSettings({
      shortcuts: {
        winV: Boolean(value),
      },
    });
  };

  const buildConflicts = (
    key: ShortcutRecorderKey,
  ): ShortcutRecorderConflict[] => {
    if (key === "openClipboard") {
      return [
        {
          label: translatePreferenceSetting(
            preferenceT,
            SHORTCUT_RECORDER_SETTINGS.openPreference,
            "title",
          ),
          value: settings.shortcuts.openPreference,
        },
      ];
    }

    if (key === "openPreference") {
      return [
        {
          label: translatePreferenceSetting(
            preferenceT,
            SHORTCUT_RECORDER_SETTINGS.openClipboard,
            "title",
          ),
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
        const setting = SHORTCUT_RECORDER_SETTINGS[key];

        const handleChange = async (value: string) => {
          await handleShortcutChange(key, value);
        };

        return (
          <PreferenceSettingCard compact={isWin} key={key} setting={setting}>
            <ShortcutRecorder
              conflicts={buildConflicts(key)}
              onChange={handleChange}
              placeholder={translatePreferencePlaceholder(preferenceT, setting)}
              value={settings.shortcuts[key]}
            />
          </PreferenceSettingCard>
        );
      })}

      {isWin && WIN_V_SETTING ? (
        <PreferenceSettingCard compact setting={WIN_V_SETTING}>
          <SwitchControl
            disabled={false}
            onChange={handlePreferenceSettingChange}
            setting={WIN_V_SETTING}
            value={settings.shortcuts.winV}
          />
        </PreferenceSettingCard>
      ) : null}
    </OnboardingStepLayout>
  );
};

export default ShortcutsStep;
