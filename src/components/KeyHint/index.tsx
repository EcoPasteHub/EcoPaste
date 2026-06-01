import { useEventListener, useKeyPress } from "ahooks";
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
   * 默认渲染的内容（通常是图标），修饰键未按下时显示。
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
  const { hintKey, onKeyPress, children, className } = props;

  const [active, setActive] = useState(false);

  const handleKeyDown = (event: KeyboardEvent) => {
    const isModifierPressed = isMac ? event.metaKey : event.ctrlKey;

    if (!isModifierPressed) return;

    setActive(true);
  };

  const handleKeyUp = () => {
    setActive(false);
  };

  const handleBlur = () => {
    setActive(false);
  };

  useKeyboardEvent("keydown", handleKeyDown);
  useKeyboardEvent("keyup", handleKeyUp);

  useEventListener("blur", handleBlur, { target: window });

  const modifierKey = isMac ? "meta" : "ctrl";

  useKeyPress(
    `${modifierKey}.${hintKey.toLowerCase()}`,
    (event) => {
      event.preventDefault();
      onKeyPress?.(event);
    },
    { exactMatch: true },
  );

  return (
    <div className="relative">
      {active && (
        <span
          className={cn(
            "-translate-1/2 absolute top-1/2 left-1/2 inline-flex size-4 items-center justify-center rounded-1 bg-black font-bold font-mono text-3! text-white",
            className,
          )}
        >
          {hintKey.toUpperCase()}
        </span>
      )}

      {children}
    </div>
  );
};

export default KeyHint;
