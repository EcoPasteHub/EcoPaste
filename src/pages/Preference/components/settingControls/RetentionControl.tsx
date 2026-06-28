import { InputNumber, Space } from "antd";
import type { FC } from "react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import type { RetentionUnit } from "@/types/settings";
import type {
  PreferenceSetting,
  RetentionSettingValue,
  SettingValue,
} from "../../types/preferences";
import type { ControlProps } from "./types";

const DEFAULT_RETENTION_UNIT: RetentionUnit = "days";

interface RetentionControlProps extends ControlProps {
  setting: PreferenceSetting;
  value: RetentionSettingValue;
}

/**
 * 将未知设置值归一成保留周期对象，避免空快照或 schema 误配导致控件崩溃。
 */
export function resolveRetentionValue(
  value?: SettingValue,
): RetentionSettingValue {
  if (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value) &&
    "unit" in value &&
    "value" in value
  ) {
    return value;
  }

  return { unit: DEFAULT_RETENTION_UNIT, value: 0 };
}

/**
 * 历史保留周期控件：单位固定为天，0 表示不按时间清理。
 */
const RetentionControl: FC<RetentionControlProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { disabled, onChange, setting, value } = props;
  const [draftValue, setDraftValue] = useState<number | null>(value.value);

  useEffect(() => {
    setDraftValue(value.value);
  }, [value]);

  const commit = async (nextValue: number | null) => {
    const normalizedValue = Math.max(0, nextValue ?? 0);
    const next: RetentionSettingValue = {
      unit: DEFAULT_RETENTION_UNIT,
      value: normalizedValue,
    };

    setDraftValue(normalizedValue);
    await onChange(setting, next);
  };

  const handleValueChange = (next: number | null) => {
    setDraftValue(next);
  };

  const handleBlur = async () => {
    await commit(draftValue);
  };

  const handlePressEnter = async () => {
    await commit(draftValue);
  };

  return (
    <Space.Compact>
      <InputNumber
        disabled={disabled}
        min={0}
        onBlur={handleBlur}
        onChange={handleValueChange}
        onPressEnter={handlePressEnter}
        value={draftValue}
      />
      <Space.Addon>{t("schema.retentionUnits.days")}</Space.Addon>
    </Space.Compact>
  );
};

export default RetentionControl;
