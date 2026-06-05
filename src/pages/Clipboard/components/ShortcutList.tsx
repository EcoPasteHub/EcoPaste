import type { FC, ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { isMac } from "@/utils/is";

interface Shortcut {
  labelKey: string;
  /** 按下顺序排列的按键，每个元素渲染为一个 kbd 徽标。 */
  keys: ReactNode[];
}

/**
 * 单个按键徽标：灰底圆角小方块，配合 monospace 显示符号或字母。
 */
const Kbd: FC<{ children: ReactNode }> = (props) => {
  const { children } = props;

  return (
    <span className="inline-flex h-6 min-w-6 items-center justify-center rounded-1.5 bg-ant-fill-secondary px-1.5 font-mono text-ant-secondary text-xs">
      {children}
    </span>
  );
};

/** 平台修饰键符号：macOS 用 ⌘，Windows 用文字。 */
const MOD = isMac ? "⌘" : "Ctrl";
const ENTER = isMac ? "⏎" : "Enter";
const BACKSPACE = "⌫";
const UP = "↑";
const DOWN = "↓";

const SHORTCUTS: Shortcut[] = [
  { keys: [ENTER], labelKey: "shortcuts.pasteSelected" },
  { keys: [MOD, ENTER], labelKey: "shortcuts.pasteSelectedPlain" },
  { keys: [MOD, BACKSPACE], labelKey: "shortcuts.deleteSelected" },
  { keys: [MOD, "D"], labelKey: "shortcuts.favoriteSelected" },
  { keys: [UP, "/", DOWN], labelKey: "shortcuts.navigate" },
  { keys: [MOD, "N"], labelKey: "shortcuts.pasteNth" },
  { keys: [MOD, "F"], labelKey: "shortcuts.focusSearch" },
  { keys: [MOD, "P"], labelKey: "shortcuts.pinWindow" },
  { keys: [MOD, ","], labelKey: "shortcuts.openPreference" },
];

// 暂未实现但占位，方便后续补齐（注释提示，不渲染）：
// SHIFT 触发的组合：⇧⌘1 粘贴第一行为纯文本等。

/**
 * 快捷键速查面板：列出当前主窗口已支持的所有键盘操作；
 * 仅渲染纯文本与按键徽标，不绑定真实事件。
 */
const ShortcutList: FC = () => {
  const { t } = useTranslation("clipboard");

  return (
    <div className="flex w-72 flex-col gap-1">
      {SHORTCUTS.map((item) => (
        <div
          className="flex items-center justify-between gap-3 px-1 py-1"
          key={item.labelKey}
        >
          <span className="text-sm">{t(item.labelKey)}</span>

          <div className="flex items-center gap-1">
            {item.keys.map((key, index) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: 静态序列且不会重排
              <Kbd key={index}>{key}</Kbd>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
};

export default ShortcutList;
