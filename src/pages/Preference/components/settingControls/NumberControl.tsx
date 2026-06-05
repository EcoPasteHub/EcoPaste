import { Input, InputNumber, Space } from "antd";
import type { FC } from "react";
import { useEffect, useState } from "react";
import type { PreferenceSetting } from "../../types/preferences";
import type { ControlProps } from "./types";

interface NumberControlProps extends ControlProps {
  setting: PreferenceSetting;
  value: number;
}

/**
 * 数字输入失焦保存，避免连续输入时频繁落盘。
 */
const NumberControl: FC<NumberControlProps> = (props) => {
  const { disabled, onChange, setting, value } = props;
  const [draft, setDraft] = useState<number | null>(value);
  const control = setting.control.type === "number" ? setting.control : null;

  useEffect(() => {
    setDraft(value);
  }, [value]);

  if (!control) return null;

  const handleChange = (next: number | null) => {
    setDraft(next);
  };

  const commit = async () => {
    const min = control.min ?? 0;
    const next = typeof draft === "number" ? Math.max(min, draft) : min;
    setDraft(next);
    await onChange(setting, next);
  };

  const handleBlur = async () => {
    await commit();
  };

  const handlePressEnter = async () => {
    await commit();
  };

  if (!control.suffix) {
    return (
      <InputNumber
        className="h-8 w-28"
        disabled={disabled}
        min={control.min}
        onBlur={handleBlur}
        onChange={handleChange}
        onPressEnter={handlePressEnter}
        value={draft}
      />
    );
  }

  return (
    <Space.Compact className="h-8 w-28">
      <InputNumber
        className="h-8 flex-1"
        disabled={disabled}
        min={control.min}
        onBlur={handleBlur}
        onChange={handleChange}
        onPressEnter={handlePressEnter}
        value={draft}
      />
      <Input
        className="h-8 w-12"
        disabled={disabled}
        readOnly
        value={control.suffix}
      />
    </Space.Compact>
  );
};

export default NumberControl;
