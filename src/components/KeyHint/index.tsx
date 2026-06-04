import { useEventListener } from "ahooks";
import { type FC, type ReactNode, useState } from "react";
import { useKeyboardEvent } from "@/hooks/useKeyboardEvent";
import { cn } from "@/utils/cn";
import { isMac } from "@/utils/is";

interface KeyHintProps {
  /**
   * 徽标自定义类名。
   */
  className?: string;
  /**
   * 默认展示的 UnoCSS 图标类名（如 "i-lets-icons:pin"），存在时优先于 children 渲染，
   * 默认尺寸 text-4，需要其它尺寸时拼到类名里（如 "i-lets-icons:pin text-5"）。
   */
  iconName?: string;
  /**
   * 默认渲染的内容（通常是图标），修饰键未按下时显示；iconName 存在时被忽略。
   */
  children?: ReactNode;
  /**
   * 修饰键按下时显示的按键文案（通常是单个字母，如 "F"），同时作为组合键的主键。
   */
  hintKey: string;
  /**
   * 按下完整组合键（macOS: ⌘+key / Windows: Ctrl+key）时触发；事件默认会被 preventDefault。
   */
  onKeyPress?: (event: KeyboardEvent) => void;
}

/**
 * 通用的快捷键提示包装：默认渲染 `children`（图标等），当用户按下修饰键时，
 * 切换为一个黑底白字的小徽标展示 `hintKey`；按下完整组合键（macOS: ⌘+key / Windows: Ctrl+key）
 * 时调用 `onKeyPress`，调用方无需再额外注册一份 `useKeyPress`。
 */
const KeyHint: FC<KeyHintProps> = (props) => {
  const { hintKey, onKeyPress, iconName, children, className } = props;

  const [active, setActive] = useState(false);

  /**
   * 修饰键按下时展示快捷键提示；完整组合键命中时触发业务回调。
   */
  const handleKeyDown = (event: KeyboardEvent) => {
    const isModifierPressed = isMac ? event.metaKey : event.ctrlKey;

    if (!isModifierPressed) return;

    setActive(true);

    if (event.key.toLowerCase() !== hintKey.toLowerCase()) return;

    event.preventDefault();

    onKeyPress?.(event);
  };

  /**
   * 仅在修饰键真正松开后隐藏提示，避免按住 Ctrl/⌘ 再敲字母时闪烁。
   */
  const handleKeyUp = (event: KeyboardEvent) => {
    const isModifierPressed = isMac ? event.metaKey : event.ctrlKey;

    if (isModifierPressed) return;

    setActive(false);
  };

  const handleBlur = () => {
    setActive(false);
  };

  useKeyboardEvent("keydown", handleKeyDown);

  useKeyboardEvent("keyup", handleKeyUp);

  useEventListener("blur", handleBlur, { target: window });

  return (
    <div className="relative">
      <div className="flex items-center justify-center">
        {iconName ? <i className={cn("text-4", iconName)} /> : children}
      </div>

      {active && (
        <span
          className={cn(
            "-translate-1/2 absolute top-1/2 left-1/2 inline-flex size-4 items-center justify-center rounded-1 bg-text font-bold font-mono text-3! text-light-solid",
            className,
          )}
        >
          {hintKey.toUpperCase()}
        </span>
      )}
    </div>
  );
};

export default KeyHint;
