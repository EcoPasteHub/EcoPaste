import type { KeyboardEvent as ReactKeyboardEvent } from "react";
import { isMac } from "@/utils/is";

export type ShortcutPattern = string | readonly string[];

interface ShortcutKey {
  eventKey: string;
  shortcutKey: string;
}

const MODIFIER_EVENT_KEYS = ["Shift", "Control", "Alt", "Meta"] as const;

const MODIFIER_SHORTCUT_KEY: Record<string, string> = {
  Alt: "Alt",
  Control: "Control",
  Meta: "Command",
  Shift: "Shift",
};

const NAMED_SHORTCUT_KEY: Record<string, string> = {
  ArrowDown: "ArrowDown",
  ArrowLeft: "ArrowLeft",
  ArrowRight: "ArrowRight",
  ArrowUp: "ArrowUp",
  Backquote: "`",
  Backslash: "\\",
  Backspace: "Backspace",
  BracketLeft: "[",
  BracketRight: "]",
  Comma: ",",
  Delete: "Delete",
  End: "End",
  Enter: "Enter",
  Equal: "=",
  Escape: "Esc",
  Home: "Home",
  Insert: "Insert",
  Minus: "-",
  PageDown: "PageDown",
  PageUp: "PageUp",
  Period: ".",
  Quote: "'",
  Semicolon: ";",
  Slash: "/",
  Space: "Space",
  Tab: "Tab",
};

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

/**
 * 归一化快捷键字面量，供冲突检测忽略大小写和多余空白。
 */
export const normalizeShortcutValue = (value: string) => {
  return value
    .split("+")
    .map((key) => {
      return key.trim().toLowerCase();
    })
    .filter((key) => {
      return key !== "";
    })
    .join("+");
};

/**
 * 判断键盘事件是否来自单独按下的修饰键。
 */
export const isShortcutModifierEventKey = (eventKey: string) => {
  return MODIFIER_EVENT_KEYS.some((modifierKey) => {
    return modifierKey === eventKey;
  });
};

/**
 * 把 KeyboardEvent 映射为 Tauri global shortcut 能解析的按键片段。
 */
export const resolveShortcutEventKey = (
  event: KeyboardEvent | ReactKeyboardEvent,
) => {
  const { code, key } = event;

  if (isShortcutModifierEventKey(key)) {
    return {
      eventKey: key,
      shortcutKey: MODIFIER_SHORTCUT_KEY[key],
    };
  }

  if (/^Key[A-Z]$/.test(code)) {
    return {
      eventKey: code,
      shortcutKey: code.slice(3),
    };
  }

  if (/^Digit[0-9]$/.test(code)) {
    return {
      eventKey: code,
      shortcutKey: code.slice(5),
    };
  }

  if (isShortcutFunctionKey(key)) {
    return {
      eventKey: key,
      shortcutKey: key,
    };
  }

  const namedKey = NAMED_SHORTCUT_KEY[code] ?? NAMED_SHORTCUT_KEY[key];
  if (!namedKey) return null;

  return {
    eventKey: code,
    shortcutKey: namedKey,
  };
};

/**
 * 生成当前键盘事件对应的完整快捷键组合，按平台习惯稳定修饰键顺序。
 */
export const buildShortcutFromEvent = (
  event: KeyboardEvent | ReactKeyboardEvent,
) => {
  const primaryKey = resolveShortcutEventKey(event);

  if (!primaryKey) return null;

  if (isShortcutModifierEventKey(primaryKey.eventKey)) {
    return null;
  }

  const modifiers = resolvePressedShortcutModifiers(event, primaryKey);
  const shortcutKeys = [...modifiers, primaryKey.shortcutKey];

  if (isShortcutFunctionKey(primaryKey.shortcutKey)) {
    return shortcutKeys.join("+");
  }

  if (modifiers.length === 0) return null;

  return shortcutKeys.join("+");
};

/**
 * 生成录入态预览用的快捷键组合；允许仅包含修饰键，便于按下即显示。
 */
export const buildShortcutPreviewFromEvent = (
  event: KeyboardEvent | ReactKeyboardEvent,
) => {
  const primaryKey = resolveShortcutEventKey(event);

  if (!primaryKey) return null;

  const modifiers = resolvePressedShortcutModifiers(event, primaryKey);

  if (isShortcutModifierEventKey(primaryKey.eventKey)) {
    return [...modifiers, primaryKey.shortcutKey].join("+");
  }

  return [...modifiers, primaryKey.shortcutKey].join("+");
};

/**
 * 按当前平台习惯格式化录入中的快捷键文案。
 */
export const formatRecordedShortcut = (value: string) => {
  if (!value) return "";

  return getShortcutKeyDisplays(value).join(isMac ? "" : "+");
};

/**
 * 判断录入结果是否满足全局快捷键的最低可用组合要求。
 */
export const isRecordableShortcut = (value: string) => {
  if (!value) return false;

  const parts = value.split("+");
  const primaryKey = parts[parts.length - 1];

  if (!primaryKey) return false;

  if (isShortcutFunctionKey(primaryKey)) {
    return true;
  }

  return parts.length > 1;
};

/**
 * 根据按键状态生成修饰键数组，避免把主键重复当作修饰键写入。
 */
const resolvePressedShortcutModifiers = (
  event: KeyboardEvent | ReactKeyboardEvent,
  primaryKey: ShortcutKey,
) => {
  const modifiers: string[] = [];

  pushShortcutModifier(modifiers, event.metaKey, "Command", primaryKey);
  pushShortcutModifier(modifiers, event.ctrlKey, "Control", primaryKey);
  pushShortcutModifier(modifiers, event.altKey, "Alt", primaryKey);
  pushShortcutModifier(modifiers, event.shiftKey, "Shift", primaryKey);

  return modifiers;
};

/**
 * 在修饰键确实参与组合时写入，单独按修饰键不会形成快捷键。
 */
const pushShortcutModifier = (
  modifiers: string[],
  pressed: boolean,
  shortcutKey: string,
  primaryKey: ShortcutKey,
) => {
  if (!pressed) return;
  if (primaryKey.shortcutKey === shortcutKey) return;

  modifiers.push(shortcutKey);
};

/**
 * 判断快捷键主键是否为可独立录入的功能键。
 */
const isShortcutFunctionKey = (key: string) => {
  return /^F([1-9]|1[0-2])$/.test(key);
};
