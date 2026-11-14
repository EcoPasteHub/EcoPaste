import type { PreferenceSettingChangeHandler } from "../../types/preferences";

export interface ControlProps {
  disabled: boolean;
  onChange: PreferenceSettingChangeHandler;
}
