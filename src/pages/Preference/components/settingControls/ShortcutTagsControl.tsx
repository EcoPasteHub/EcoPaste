import type { FC } from "react";
import type { PreferenceSetting } from "../../types/preferences";

interface ShortcutTagsControlProps {
  setting: PreferenceSetting;
}

/**
 * 展示只读快捷键组合；主窗口内快捷键由代码固定，不在偏好页修改。
 */
const ShortcutTagsControl: FC<ShortcutTagsControlProps> = (props) => {
  const { setting } = props;

  if (setting.control.type !== "shortcutTags") return null;

  return (
    <div className="flex max-w-78 flex-col items-end gap-1.5">
      {setting.control.shortcuts.map((shortcut) => {
        return (
          <div className="flex items-center gap-1" key={shortcut.label}>
            {shortcut.keys.map((key) => {
              return (
                <span
                  className="inline-flex h-6 min-w-6 items-center justify-center rounded-1.5 bg-ant-fill-secondary px-1.5 font-mono text-ant-secondary text-xs leading-none"
                  key={`${shortcut.label}-${key}`}
                >
                  {key}
                </span>
              );
            })}
          </div>
        );
      })}
    </div>
  );
};

export default ShortcutTagsControl;
