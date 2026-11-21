import type { TFunction } from "i18next";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import ShortcutRecorder, {
  type ShortcutRecorderConflict,
} from "@/components/ShortcutRecorder";
import type { Settings } from "@/types/settings";
import type { PreferenceSetting } from "../../types/preferences";
import { translatePreferencePlaceholder } from "../../utils/preferenceI18n";
import type { ControlProps } from "./types";

interface ShortcutRecorderControlProps extends ControlProps {
  setting: PreferenceSetting;
  settings: Settings;
  value: string;
}

/**
 * 偏好设置里的全局快捷键录入控件，复用通用 ShortcutRecorder 并在录入完成后持久化设置。
 */
const ShortcutRecorderControl: FC<ShortcutRecorderControlProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { disabled, onChange, setting, settings, value } = props;

  if (setting.control.type !== "shortcutRecorder") return null;

  const conflicts = resolveGlobalShortcutConflicts(t, setting, settings);

  /**
   * 录入完成后把快捷键字面量交给偏好设置统一持久化流程。
   */
  const handleChange = async (nextValue: string) => {
    await onChange(setting, nextValue);
  };

  return (
    <ShortcutRecorder
      conflicts={conflicts}
      disabled={disabled}
      onChange={handleChange}
      placeholder={translatePreferencePlaceholder(t, setting)}
      value={value}
    />
  );
};

export default ShortcutRecorderControl;

/**
 * 偏好设置里两个全局快捷键必须互斥，避免同一个组合键触发多个动作。
 */
function resolveGlobalShortcutConflicts(
  t: TFunction<"preferences">,
  setting: PreferenceSetting,
  settings: Settings,
) {
  const openClipboardConflict = {
    label: translateShortcutSettingTitle(t, "shortcuts.openClipboard"),
    value: settings.shortcuts.openClipboard,
  };
  const openPreferenceConflict = {
    label: translateShortcutSettingTitle(t, "shortcuts.openPreference"),
    value: settings.shortcuts.openPreference,
  };

  switch (setting.id) {
    case "shortcuts.openClipboard":
      return [openPreferenceConflict];
    case "shortcuts.openPreference":
      return [openClipboardConflict];
    default:
      return [] satisfies ShortcutRecorderConflict[];
  }
}

/**
 * 用稳定设置 id 取快捷键设置标题，作为冲突 toast 的占用方名称。
 */
function translateShortcutSettingTitle(
  t: TFunction<"preferences">,
  id: string,
) {
  return t(`schema.settings.${id}.title`);
}
