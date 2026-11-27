import { Space, Tag } from "antd";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import type { PreferenceSetting } from "../../types/preferences";
import { translatePreferenceShortcutLabel } from "../../utils/preferenceI18n";

interface ShortcutTagsControlProps {
  setting: PreferenceSetting;
}

/**
 * 展示只读快捷键组合；剪贴板窗口内快捷键由代码固定，不在偏好页修改。
 */
const ShortcutTagsControl: FC<ShortcutTagsControlProps> = (props) => {
  const { setting } = props;
  const { t } = useTranslation("preferences");

  if (setting.control.type !== "shortcutTags") return null;

  return (
    <Space orientation="vertical">
      {setting.control.shortcuts.map((shortcut, shortcutIndex) => {
        const label = translatePreferenceShortcutLabel(
          t,
          setting,
          shortcutIndex,
        );

        return (
          <Space key={label}>
            {shortcut.keys.map((key) => {
              return <Tag key={`${label}-${key}`}>{key}</Tag>;
            })}
          </Space>
        );
      })}
    </Space>
  );
};

export default ShortcutTagsControl;
