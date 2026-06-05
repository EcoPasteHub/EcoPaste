import { Input } from "antd";
import type { ChangeEvent, FC, FocusEvent } from "react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import type { PreferenceSetting } from "../../types/preferences";
import { translatePreferencePlaceholder } from "../../utils/preferenceI18n";
import type { ControlProps } from "./types";

interface TextareaControlProps extends ControlProps {
  setting: PreferenceSetting;
  value: string[];
}

/**
 * 多行规则输入：一行一个值，失焦后整体替换数组。
 */
const TextareaControl: FC<TextareaControlProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { disabled, onChange, setting, value } = props;
  const [draft, setDraft] = useState(value.join("\n"));

  const placeholder =
    setting.control.type === "textarea"
      ? translatePreferencePlaceholder(t, setting)
      : "";

  useEffect(() => {
    setDraft(value.join("\n"));
  }, [value]);

  const handleChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    setDraft(event.target.value);
  };

  const handleBlur = async (_event: FocusEvent<HTMLTextAreaElement>) => {
    const next = draft
      .split("\n")
      .map((line) => {
        return line.trim();
      })
      .filter((line) => {
        return line.length > 0;
      });
    await onChange(setting, next);
  };

  return (
    <Input.TextArea
      autoSize={{ maxRows: 5, minRows: 2 }}
      className="w-76"
      disabled={disabled}
      onBlur={handleBlur}
      onChange={handleChange}
      placeholder={placeholder}
      value={draft}
    />
  );
};

export default TextareaControl;
