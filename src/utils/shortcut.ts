import { isMac } from "@/utils/is";

export type ShortcutPattern = string | readonly string[];

const KEY_DISPLAY: Record<string, string> = {
  Alt: isMac ? "⌥" : "Alt",
  ArrowDown: "↓",
  ArrowLeft: "←",
  ArrowRight: "→",
  ArrowUp: "↑",
  Backquote: "`",
  Backslash: "\\",
  Backspace: "⌫",
  Cmd: isMac ? "⌘" : "Ctrl",
  CmdOrCtrl: isMac ? "⌘" : "Ctrl",
  Comma: ",",
  Command: isMac ? "⌘" : "Win",
  CommandOrControl: isMac ? "⌘" : "Ctrl",
  Control: isMac ? "⌃" : "Ctrl",
  Ctrl: isMac ? "⌃" : "Ctrl",
  Delete: isMac ? "⌦" : "Del",
  Down: "↓",
  Enter: isMac ? "⏎" : "Enter",
  Equal: "=",
  Esc: isMac ? "⎋" : "Esc",
  Escape: isMac ? "⎋" : "Esc",
  Left: "←",
  Meta: isMac ? "⌘" : "Win",
  Minus: "-",
  Mod: isMac ? "⌘" : "Ctrl",
  Option: isMac ? "⌥" : "Alt",
  Period: ".",
  Quote: "'",
  Return: isMac ? "⏎" : "Enter",
  Right: "→",
  Semicolon: ";",
  Shift: isMac ? "⇧" : "Shift",
  Slash: "/",
  Space: isMac ? "␣" : "Space",
  Super: isMac ? "⌘" : "Win",
  Tab: isMac ? "⇥" : "Tab",
  Up: "↑",
};

/**
 * 把单个快捷键标识转成当前平台适合展示的文案或符号。
 */
export const getShortcutKeyDisplay = (key: string) => {
  const normalizedKey = key.trim();
  const display = KEY_DISPLAY[normalizedKey];

  if (display) return display;

  if (/^Key[A-Z]$/.test(normalizedKey)) {
    return normalizedKey.slice(3);
  }

  if (/^Digit[0-9]$/.test(normalizedKey)) {
    return normalizedKey.slice(5);
  }

  if (normalizedKey.length === 1) {
    return normalizedKey.toUpperCase();
  }

  return normalizedKey;
};

/**
 * 把快捷键组合拆成当前平台的按键展示数组，供 kbd 徽标逐个渲染。
 */
export const getShortcutKeyDisplays = (shortcut: ShortcutPattern) => {
  const keys: readonly string[] =
    typeof shortcut === "string" ? shortcut.split("+") : shortcut;

  return keys.map((key) => {
    return getShortcutKeyDisplay(key);
  });
};

/**
 * 把快捷键组合格式化成当前平台的一行展示文案。
 */
export const formatShortcutDisplay = (
  shortcut: ShortcutPattern,
  separator = " + ",
) => {
  return getShortcutKeyDisplays(shortcut).join(separator);
};
