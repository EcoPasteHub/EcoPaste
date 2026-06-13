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
  value: string | number;
}

export interface PreferenceShortcutTag {
  keys: string[];
}

export type PreferenceControl =
  | { type: "switch" }
  | { type: "segmented"; options: PreferenceOption[] }
  | { type: "select"; options: PreferenceOption[]; mode?: "multiple" }
  | { type: "clipboardGroupSelect" }
  | {
      type: "sortableTree";
      options: PreferenceOption[];
    }
  | {
      type: "sortableCheckboxTree";
      options: PreferenceOption[];
      orderPath: readonly string[];
    }
  | {
      type: "number";
      max?: number;
      min?: number;
      suffixKey?: string;
    }
  | { type: "retention" }
  | { type: "text" }
  | { type: "shortcutRecorder" }
  | { type: "textarea" }
  | { type: "appExclusion" }
  | { type: "action"; danger?: boolean }
  | { type: "status" }
  | { type: "shortcutTags"; shortcuts: PreferenceShortcutTag[] };

export interface PreferenceSetting {
  control: PreferenceControl;
  disabled?: boolean;
  disabledWhen?: (settings: Settings) => boolean;
  id: string;
  keywords?: string[];
  parentId?: string;
  path?: readonly string[];
  status?: "comingSoon" | "alwaysOn" | "requiresBackend" | "experimental";
  value?: (settings: Settings) => SettingValue;
}

export type PreferenceSettingChangeHandler = (
  setting: PreferenceSetting,
  value: SettingValue,
) => Promise<void>;

export interface PreferenceSection {
  id: string;
  settings: PreferenceSetting[];
}

export interface PreferenceTab {
  icon: string;
  id: PreferenceTabId;
  sections: PreferenceSection[];
}
