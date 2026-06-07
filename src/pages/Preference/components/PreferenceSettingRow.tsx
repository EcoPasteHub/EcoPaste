import type { TFunction } from "i18next";
import { motion } from "motion/react";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import type {
  ChangeStorageLocationResult,
  CleanCacheResult,
  ExportHistoryBackupResult,
  StorageLocation,
} from "@/commands";
import type { Settings } from "@/types/settings";
import { cn } from "@/utils/cn";
import type {
  PreferenceSetting,
  PreferenceSettingChangeHandler,
  SettingValue,
} from "../types/preferences";
import { translatePreferenceSetting } from "../utils/preferenceI18n";
import PreferenceStatusBadge from "./PreferenceStatusBadge";
import ActionControl from "./settingControls/ActionControl";
import NumberControl from "./settingControls/NumberControl";
import RetentionControl, {
  resolveRetentionValue,
} from "./settingControls/RetentionControl";
import {
  SegmentedSelectControl,
  SelectControl,
} from "./settingControls/SelectControls";
import ShortcutTagsControl from "./settingControls/ShortcutTagsControl";
import SortableCheckboxTreeControl from "./settingControls/SortableCheckboxTreeControl";
import StatusControl from "./settingControls/StatusControl";
import SwitchControl from "./settingControls/SwitchControl";
import { resolveSettingVisual } from "./settingControls/settingVisual";
import TextareaControl from "./settingControls/TextareaControl";
import TextControl from "./settingControls/TextControl";

interface PreferenceSettingRowProps {
  highlighted: boolean;
  highlightToken: number;
  setting: PreferenceSetting;
  settings: Settings;
  shouldReduceMotion: boolean;
  storageLocation: StorageLocation | null;
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
 * 单个设置项行：左侧说明结果，右侧渲染对应控件。
 */
const PreferenceSettingRow: FC<PreferenceSettingRowProps> = (props) => {
  const { t } = useTranslation("preferences");
  const {
    highlighted,
    highlightToken,
    setting,
    settings,
    shouldReduceMotion,
    storageLocation,
    onActionComplete,
    onChange,
  } = props;
  const value = setting.value?.(settings);
  const parentDisabled = setting.disabledWhen?.(settings) === true;
  const disabled = setting.disabled === true || parentDisabled;
  const childSetting = setting.parentId !== void 0;
  const collapsed = childSetting && parentDisabled;
  const visual = resolveSettingVisual(setting.id);
  const highlightOpacity = shouldReduceMotion
    ? 0.08
    : [0, 0.1, 0.055, 0.085, 0.045, 0.07, 0.025, 0];
  const rowTransition = {
    duration: shouldReduceMotion ? 0 : 0.16,
    ease: "easeOut",
  } as const;

  return (
    <motion.div
      animate={{
        borderBottomWidth: collapsed ? 0 : 1,
        height: collapsed ? 0 : "auto",
        minHeight: collapsed ? 0 : "3.875rem",
        opacity: collapsed ? 0 : 1,
        paddingBottom: collapsed ? 0 : "0.6875rem",
        paddingTop: collapsed ? 0 : "0.6875rem",
      }}
      className={cn(
        "group relative flex items-center gap-5 overflow-hidden border-ant-split border-b px-4 transition-colors last:border-b-0 hover:bg-ant-fill-quaternary motion-reduce:transition-none",
        { "pl-12": childSetting, "pointer-events-none": collapsed },
      )}
      data-preference-setting-id={setting.id}
      initial={false}
      transition={rowTransition}
    >
      {highlighted ? (
        <motion.span
          animate={{ opacity: highlightOpacity }}
          className="pointer-events-none absolute inset-0 bg-ant-primary"
          initial={{ opacity: 0 }}
          key={highlightToken}
          transition={{
            duration: shouldReduceMotion ? 0 : 2.35,
            ease: "easeInOut",
            times: shouldReduceMotion
              ? void 0
              : [0, 0.11, 0.28, 0.43, 0.58, 0.73, 0.88, 1],
          }}
        />
      ) : null}

      <span
        className={cn(
          "absolute top-3 bottom-3 left-0 w-0.5 rounded-full transition-colors motion-reduce:transition-none",
          highlighted
            ? "bg-ant-primary"
            : "bg-transparent group-hover:bg-ant-primary",
        )}
      />

      <span
        className={cn(
          "relative flex size-7.5 shrink-0 items-center justify-center text-xl transition-colors motion-reduce:transition-none",
          disabled ? "text-ant-disabled" : visual.tone,
        )}
      >
        <i aria-hidden="true" className={visual.icon} />
      </span>

      <div className="relative min-w-0 flex-1">
        <div
          className={cn("flex min-w-0 items-center gap-2", {
            "opacity-65": disabled,
          })}
        >
          <span
            className={cn(
              "truncate font-medium text-sm leading-snug",
              disabled ? "text-ant-secondary" : "text-ant-text",
            )}
          >
            {translatePreferenceSetting(t, setting, "title")}
          </span>
          <PreferenceStatusBadge compact status={setting.status} />
        </div>
        <div
          className={cn(
            "mt-0.5 text-xs leading-snug",
            disabled ? "text-ant-disabled" : "text-ant-tertiary",
            { "break-all": setting.id === "localData.dataDirectory" },
          )}
        >
          {resolveSettingDescription(t, setting, storageLocation)}
        </div>
      </div>

      <div className="relative flex shrink-0 justify-end opacity-90 transition-opacity group-hover:opacity-100 motion-reduce:transition-none">
        {renderControl(
          setting,
          disabled,
          onChange,
          value,
          onActionComplete,
          storageLocation,
        )}
      </div>
    </motion.div>
  );
};

export default PreferenceSettingRow;

/**
 * 根据 schema 中声明的控件类型分发到具体设置控件。
 */
function renderControl(
  setting: PreferenceSetting,
  disabled: boolean,
  onChange: PreferenceSettingChangeHandler,
  value?: SettingValue,
  onActionComplete?: (
    setting: PreferenceSetting,
    result?:
      | ChangeStorageLocationResult
      | CleanCacheResult
      | ExportHistoryBackupResult,
  ) => void,
  storageLocation?: StorageLocation | null,
) {
  switch (setting.control.type) {
    case "switch":
      return (
        <SwitchControl
          disabled={disabled}
          onChange={onChange}
          setting={setting}
          value={Boolean(value)}
        />
      );
    case "segmented":
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
    case "sortableCheckboxTree":
      return (
        <SortableCheckboxTreeControl
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
}

/**
 * 数据目录行展示当前真实路径，其它设置沿用 schema 文案。
 */
function resolveSettingDescription(
  t: TFunction<"preferences">,
  setting: PreferenceSetting,
  storageLocation: StorageLocation | null,
) {
  if (setting.id === "localData.dataDirectory" && storageLocation) {
    return storageLocation.currentPath;
  }

  return translatePreferenceSetting(t, setting, "description");
}
