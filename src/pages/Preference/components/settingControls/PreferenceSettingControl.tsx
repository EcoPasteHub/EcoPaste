import type { FC } from "react";
import type {
  ChangeStorageLocationResult,
  CleanCacheResult,
  ExportHistoryBackupResult,
  StorageLocation,
} from "@/commands";
import LanguageSwitcher from "@/components/LanguageSwitcher";
import type { Language, Settings } from "@/types/settings";
import type {
  PreferenceSetting,
  PreferenceSettingChangeHandler,
  SettingValue,
} from "../../types/preferences";
import { SponsorQrControl } from "./AboutControls";
import ActionControl from "./ActionControl";
import CaptureOrderControl from "./CaptureOrderControl";
import ClipboardGroupSelectControl from "./ClipboardGroupSelectControl";
import NumberControl from "./NumberControl";
import PermissionControl from "./PermissionControl";
import RetentionControl, { resolveRetentionValue } from "./RetentionControl";
import { SegmentedSelectControl, SelectControl } from "./SelectControls";
import ShortcutRecorderControl from "./ShortcutRecorderControl";
import ShortcutTagsControl from "./ShortcutTagsControl";
import SortableCheckboxTreeControl from "./SortableCheckboxTreeControl";
import StatusControl from "./StatusControl";
import SwitchControl from "./SwitchControl";
import TextareaControl from "./TextareaControl";
import TextControl from "./TextControl";

interface PreferenceSettingControlProps {
  disabled: boolean;
  setting: PreferenceSetting;
  settings: Settings;
  value?: SettingValue;
  storageLocation?: StorageLocation | null;
  onActionComplete?: (
    setting: PreferenceSetting,
    result?:
      | ChangeStorageLocationResult
      | CleanCacheResult
      | ExportHistoryBackupResult,
  ) => void;
  onChange: PreferenceSettingChangeHandler;
}

/**
 * 根据 schema 声明的控件类型渲染设置控件，供偏好页和引导页共享。
 */
const PreferenceSettingControl: FC<PreferenceSettingControlProps> = (props) => {
  const {
    disabled,
    setting,
    settings,
    storageLocation,
    value,
    onActionComplete,
    onChange,
  } = props;

  const handleLanguageChange = async (nextLanguage: Language) => {
    await onChange(setting, nextLanguage);
  };

  switch (setting.control.type) {
    case "sponsorQr":
      return <SponsorQrControl setting={setting} />;
    case "switch":
      return (
        <SwitchControl
          disabled={disabled}
          onChange={onChange}
          setting={setting}
          value={Boolean(value)}
        />
      );
    case "permission":
      return <PermissionControl disabled={disabled} setting={setting} />;
    case "segmented":
      if (setting.id === "appearance.language") {
        return (
          <LanguageSwitcher
            disabled={disabled}
            onChange={handleLanguageChange}
            value={settings.appearance.language}
          />
        );
      }

      return (
        <SegmentedSelectControl
          disabled={disabled}
          onChange={onChange}
          setting={setting}
          value={String(value ?? "")}
        />
      );
    case "select":
      return (
        <SelectControl
          disabled={disabled}
          onChange={onChange}
          setting={setting}
          value={value}
        />
      );
    case "clipboardGroupSelect":
      return (
        <ClipboardGroupSelectControl
          disabled={disabled}
          onChange={onChange}
          setting={setting}
          value={typeof value === "string" ? value : "preserve"}
        />
      );
    case "sortableCheckboxTree":
      return (
        <SortableCheckboxTreeControl
          disabled={disabled}
          onChange={onChange}
          setting={setting}
          value={value}
        />
      );
    case "sortableTree":
      return (
        <CaptureOrderControl
          disabled={disabled}
          onChange={onChange}
          setting={setting}
          value={value}
        />
      );
    case "number":
      return (
        <NumberControl
          disabled={disabled}
          onChange={onChange}
          setting={setting}
          value={typeof value === "number" ? value : 0}
        />
      );
    case "retention":
      return (
        <RetentionControl
          disabled={disabled}
          onChange={onChange}
          setting={setting}
          value={resolveRetentionValue(value)}
        />
      );
    case "text":
      return (
        <TextControl
          disabled={disabled}
          onChange={onChange}
          setting={setting}
          value={typeof value === "string" ? value : ""}
        />
      );
    case "shortcutRecorder":
      return (
        <ShortcutRecorderControl
          disabled={disabled}
          onChange={onChange}
          setting={setting}
          settings={settings}
          value={typeof value === "string" ? value : ""}
        />
      );
    case "textarea":
    case "appExclusion":
      return (
        <TextareaControl
          disabled={disabled}
          onChange={onChange}
          setting={setting}
          value={Array.isArray(value) ? value : []}
        />
      );
    case "action":
      return (
        <ActionControl
          disabled={disabled}
          onActionComplete={onActionComplete}
          setting={setting}
          storageLocation={storageLocation ?? null}
        />
      );
    case "status":
      return <StatusControl setting={setting} />;
    case "shortcutTags":
      return <ShortcutTagsControl setting={setting} />;
  }
};

export default PreferenceSettingControl;
