import { InputNumber, Space } from "antd";
import type { FC } from "react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import type { PreferenceSetting } from "../../types/preferences";
import { translatePreferenceNumberSuffix } from "../../utils/preferenceI18n";
import type { ControlProps } from "./types";

interface NumberControlProps extends ControlProps {
  setting: PreferenceSetting;
  value: number;
}

/**
 * 数字输入失焦保存，避免连续输入时频繁落盘。
 */
const NumberControl: FC<NumberControlProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { disabled, onChange, setting, value } = props;
  const [draft, setDraft] = useState<number | null>(value);
  const control = setting.control.type === "number" ? setting.control : null;

  useEffect(() => {
    setDraft(value);
  }, [value]);

  if (!control) return null;

  const suffix = translatePreferenceNumberSuffix(t, setting);

  const handleChange = (next: number | null) => {
    setDraft(next);
  };

  const commit = async () => {
    const min = control.min ?? 0;
    const max = control.max ?? Number.POSITIVE_INFINITY;
    const current = typeof draft === "number" ? draft : min;
    const next = Math.min(max, Math.max(min, current));
    setDraft(next);
    await onChange(setting, next);
  };

  const handleBlur = async () => {
    await commit();
  };

  const handlePressEnter = async () => {
    await commit();
  };

  if (!suffix) {
    return (
      <InputNumber
        disabled={disabled}
        max={control.max}
        min={control.min}
        onBlur={handleBlur}
        onChange={handleChange}
        onPressEnter={handlePressEnter}
        value={draft}
      />
    );
  }

  return (
    <Space.Compact>
      <InputNumber
        disabled={disabled}
        max={control.max}
        min={control.min}
        onBlur={handleBlur}
        onChange={handleChange}
        onPressEnter={handlePressEnter}
        value={draft}
      />
      <Space.Addon>{suffix}</Space.Addon>
    </Space.Compact>
  );
};

export default NumberControl;
