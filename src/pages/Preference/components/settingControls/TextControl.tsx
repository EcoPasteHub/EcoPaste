import { Input } from "antd";
import type { ChangeEvent, FC } from "react";
import { useEffect, useState } from "react";
import type { PreferenceSetting } from "../../types/preferences";
import type { ControlProps } from "./types";

interface TextControlProps extends ControlProps {
  setting: PreferenceSetting;
  value: string;
}

/**
 * 短文本输入失焦保存，主要用于快捷键字面量。
 */
const TextControl: FC<TextControlProps> = (props) => {
  const { disabled, onChange, setting, value } = props;
  const [draft, setDraft] = useState(value);
  const control = setting.control.type === "text" ? setting.control : null;

  useEffect(() => {
    setDraft(value);
  }, [value]);

  if (!control) return null;

  const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    setDraft(event.target.value);
  };

  const commit = async () => {
    await onChange(setting, draft.trim());
  };

  const handleBlur = async () => {
    await commit();
  };

  const handlePressEnter = async () => {
    await commit();
  };

  return (
    <Input
      className="h-8 w-52"
      disabled={disabled}
      onBlur={handleBlur}
      onChange={handleChange}
      onPressEnter={handlePressEnter}
      placeholder={control.placeholder}
      value={draft}
    />
  );
};

export default TextControl;
