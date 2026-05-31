import { useEventListener, useKeyPress } from "ahooks";
import type { KeyType } from "ahooks/lib/useKeyPress";
import { type FC, type ReactNode, useState } from "react";
import { cn } from "@/utils/cn";
import { isMac } from "@/utils/is";

type Modifier = "auto" | "meta" | "ctrl" | "alt" | "shift";

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
   * 触发的修饰键。`auto` 时 macOS 用 ⌘、Windows 用 Ctrl。
   */
  modifier?: Modifier;
  /**
   * 按下完整组合键（修饰键 + hintKey）时触发；事件默认会被 preventDefault。
   */
  onKeyPress?: (event: KeyboardEvent) => void;
}

/**
 * 把 `auto` 解析为当前平台真实修饰键名。
 */
const resolveModifier = (modifier: Modifier) =>
  modifier === "auto" ? (isMac ? "meta" : "ctrl") : modifier;

/**
 * 判断键盘事件中目标修饰键是否处于按下状态。
 */
const isModifierPressed = (event: KeyboardEvent, modifier: Modifier) => {
  switch (resolveModifier(modifier)) {
    case "meta":
      return event.metaKey;
    case "ctrl":
      return event.ctrlKey;
    case "alt":
      return event.altKey;
    case "shift":
      return event.shiftKey;
  }
};

/**
 * 通用的快捷键提示包装：默认渲染 `children`（图标等），当用户按下修饰键时，
 * 切换为一个黑底白字的小徽标展示 `hintKey`；按下完整组合键（修饰键 + hintKey）
 * 时调用 `onKeyPress`，调用方无需再额外注册一份 `useKeyPress`。
 */
const KeyHint: FC<KeyHintProps> = (props) => {
  const { hintKey, modifier = "auto", onKeyPress, children, className } = props;

  const [active, setActive] = useState(false);

  useEventListener("keydown", (event) => {
    if (isModifierPressed(event, modifier)) setActive(true);
  });

  useEventListener("keyup", () => setActive(false));

  useEventListener("blur", () => setActive(false), { target: () => window });

  useKeyPress(
    `${resolveModifier(modifier)}.${hintKey.toLowerCase()}` as KeyType,
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
