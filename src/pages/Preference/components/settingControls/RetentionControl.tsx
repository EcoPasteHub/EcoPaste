import { InputNumber, Select, Space } from "antd";
import type { FC } from "react";
import { useEffect, useState } from "react";
import type { RetentionUnit } from "@/types/settings";
import type {
  PreferenceSetting,
  RetentionSettingValue,
  SettingValue,
} from "../../types/preferences";
import type { ControlProps } from "./types";

const DEFAULT_RETENTION_UNIT: RetentionUnit = "days";
const RETENTION_UNIT_OPTIONS = [
  { label: "小时", value: "hours" },
  { label: "天", value: "days" },
  { label: "周", value: "weeks" },
  { label: "月", value: "months" },
];

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

  return { unit: "forever", value: 0 };
}

/**
 * 历史保留周期组合控件：0 写为永久保留，其余数值配合单位写回。
 */
const RetentionControl: FC<RetentionControlProps> = (props) => {
  const { disabled, onChange, setting, value } = props;
  const initialUnit =
    value.unit === "forever" ? DEFAULT_RETENTION_UNIT : value.unit;
  const initialValue = value.unit === "forever" ? 0 : value.value;
  const [draftValue, setDraftValue] = useState<number | null>(initialValue);
  const [draftUnit, setDraftUnit] = useState<RetentionUnit>(initialUnit);

  useEffect(() => {
    setDraftValue(value.unit === "forever" ? 0 : value.value);
    setDraftUnit(
      value.unit === "forever" ? DEFAULT_RETENTION_UNIT : value.unit,
    );
  }, [value]);

  const commit = async (nextValue: number | null, nextUnit: RetentionUnit) => {
    const normalizedValue = Math.max(0, nextValue ?? 0);
    const normalizedUnit =
      nextUnit === "forever" ? DEFAULT_RETENTION_UNIT : nextUnit;
    const next: RetentionSettingValue =
      normalizedValue === 0
        ? { unit: "forever", value: 0 }
        : { unit: normalizedUnit, value: normalizedValue };

    setDraftValue(normalizedValue);
    setDraftUnit(normalizedUnit);
    await onChange(setting, next);
  };

  const handleValueChange = (next: number | null) => {
    setDraftValue(next);
  };

  const handleBlur = async () => {
    await commit(draftValue, draftUnit);
  };

  const handlePressEnter = async () => {
    await commit(draftValue, draftUnit);
  };

  const handleUnitChange = async (next: string | number) => {
    const nextUnit = next as RetentionUnit;
    setDraftUnit(nextUnit);
    await commit(draftValue, nextUnit);
  };

  return (
    <Space.Compact className="h-8 w-40">
      <InputNumber
        className="h-8 w-18"
        disabled={disabled}
        min={0}
        onBlur={handleBlur}
        onChange={handleValueChange}
        onPressEnter={handlePressEnter}
        value={draftValue}
      />
      <Select
        className="h-8 flex-1"
        disabled={disabled}
        onChange={handleUnitChange}
        options={RETENTION_UNIT_OPTIONS}
        value={draftUnit}
      />
    </Space.Compact>
  );
};

export default RetentionControl;
