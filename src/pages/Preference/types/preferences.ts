import type { RetentionUnit, Settings } from "@/types/settings";

export type PreferenceTabId =
  | "record"
  | "organize"
  | "reuse"
  | "workflow"
  | "shortcuts"
  | "data"
  | "about";

export interface RetentionSettingValue {
  unit: RetentionUnit;
  value: number;
}

export interface SortableCheckboxTreeSettingValue {
  order: string[];
  selected: string[];
}

export type SettingValue =
  | boolean
  | number
  | string
  | string[]
  | SortableCheckboxTreeSettingValue
  | RetentionSettingValue;

export type PreferenceStorageState = "loading" | "ready" | "error";

export interface PreferenceOption {
  label: string;
  value: string | number;
}

export interface PreferenceShortcutTag {
  label: string;
  keys: string[];
}

export type PreferenceControl =
  | { type: "switch" }
  | { type: "segmented"; options: PreferenceOption[] }
  | { type: "select"; options: PreferenceOption[]; mode?: "multiple" }
  | {
      type: "sortableTree";
      label: string;
      options: PreferenceOption[];
    }
  | {
      type: "sortableCheckboxTree";
      label: string;
      options: PreferenceOption[];
      orderPath: readonly string[];
    }
  | {
      type: "number";
      max?: number;
      min?: number;
      placeholder?: string;
      suffix?: string;
      suffixKey?: string;
    }
  | { type: "retention" }
  | { type: "text"; placeholder?: string }
  | { type: "textarea"; placeholder?: string }
  | { type: "appExclusion" }
  | { type: "action"; danger?: boolean; label: string }
  | { type: "status"; label: string }
  | { type: "shortcutTags"; shortcuts: PreferenceShortcutTag[] };

export interface PreferenceSetting {
  control: PreferenceControl;
  description: string;
  disabled?: boolean;
  disabledWhen?: (settings: Settings) => boolean;
  id: string;
  keywords?: string[];
  parentId?: string;
  path?: readonly string[];
  status?: "comingSoon" | "alwaysOn" | "requiresBackend" | "experimental";
  title: string;
  value?: (settings: Settings) => SettingValue;
}

export type PreferenceSettingChangeHandler = (
  setting: PreferenceSetting,
  value: SettingValue,
) => Promise<void>;

export interface PreferenceSection {
  description: string;
  id: string;
  settings: PreferenceSetting[];
  title: string;
}

export interface PreferenceTab {
  icon: string;
  id: PreferenceTabId;
  sections: PreferenceSection[];
  title: string;
}
