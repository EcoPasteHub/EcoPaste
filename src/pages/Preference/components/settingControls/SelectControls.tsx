import { Select } from "antd";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import type { PreferenceSetting, SettingValue } from "../../types/preferences";
import { translatePreferenceOption } from "../../utils/preferenceI18n";
import type { ControlProps } from "./types";

interface SegmentedSelectControlProps extends ControlProps {
  setting: PreferenceSetting;
  value: string;
}

/**
 * 即时保存短选项模式设置，视觉统一为 Select。
 */
export const SegmentedSelectControl: FC<SegmentedSelectControlProps> = (
  props,
) => {
  const { t } = useTranslation("preferences");
  const { disabled, onChange, setting, value } = props;

  if (setting.control.type !== "segmented") return null;

  const options = setting.control.options.map((option) => {
    return translatePreferenceOption(t, setting, option);
  });

  const handleChange = async (next: string | number) => {
    await onChange(setting, next);
  };

  return (
    <Select
      disabled={disabled}
      onChange={handleChange}
      options={options}
      value={value}
    />
  );
};

interface SelectControlProps extends ControlProps {
  setting: PreferenceSetting;
  value?: SettingValue;
}

/**
 * 即时保存长选项或多选设置。
 */
export const SelectControl: FC<SelectControlProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { disabled, onChange, setting, value } = props;

  if (setting.control.type !== "select") return null;

  const options = setting.control.options.map((option) => {
    return translatePreferenceOption(t, setting, option);
  });

  const handleChange = async (next: string | number | string[] | number[]) => {
    await onChange(setting, next as SettingValue);
  };

  return (
    <Select
      disabled={disabled}
      mode={setting.control.mode}
      onChange={handleChange}
      options={options}
      value={value as string | number | string[] | number[]}
    />
  );
};
