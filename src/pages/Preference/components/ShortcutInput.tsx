import { Button, Popover } from "@heroui/react";
import { cn } from "@heroui/styles";
import { type KeyboardEvent, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

const IS_MAC =
  typeof navigator !== "undefined" &&
  /Mac|iP(hone|ad)/.test(navigator.platform);

/**
 * 修饰键名（与 tauri-plugin-global-shortcut 接受的写法对齐）。
 * macOS 上 metaKey 映射 "Cmd"；Windows 上 metaKey（Win 键）几乎不用于全局快捷键，
 * 映射 "Super" 仅作兜底，实际几乎不会触达。
 */
const MOD_NAMES = {
  alt: IS_MAC ? "Option" : "Alt",
  ctrl: "Ctrl",
  meta: IS_MAC ? "Cmd" : "Super",
  shift: "Shift",
} as const;

/**
 * 显示时把 Option 写回 Alt（Rust 端接受 Alt；Option 仅给用户看更友好的写法）。
 * 实际写入 settings 时统一用 Alt，避免分平台两套字符串。
 */
const toCanonical = (s: string) => s.replace(/\bOption\b/g, "Alt");

interface Captured {
  mods: string[];
  key: string | null;
}

const EMPTY: Captured = { key: null, mods: [] };

/**
 * 把 KeyboardEvent.code 转成 tauri shortcut 可识别的 key 名；返回 null 表示纯修饰键。
 */
const codeToKey = (code: string): string | null => {
  if (code.startsWith("Key")) return code.slice(3);
  if (code.startsWith("Digit")) return code.slice(5);
  if (/^F\d{1,2}$/.test(code)) return code;
  if (code.startsWith("Arrow")) return code.slice(5);
  switch (code) {
    case "Space":
    case "Enter":
    case "Escape":
    case "Tab":
    case "Backspace":
    case "Delete":
    case "Home":
    case "End":
    case "PageUp":
    case "PageDown":
    case "Minus":
    case "Equal":
    case "Backslash":
    case "Slash":
    case "Comma":
    case "Period":
      return code;
    default:
      return null;
  }
};

const buildBinding = (c: Captured, modifierOnly: boolean): string => {
  const parts = c.mods.slice();
  if (!modifierOnly && c.key) parts.push(c.key);
  return parts.join("+");
};

const captureFromEvent = (e: KeyboardEvent<HTMLElement>): Captured => {
  const mods: string[] = [];
  if (e.metaKey) mods.push(MOD_NAMES.meta);
  if (e.ctrlKey) mods.push(MOD_NAMES.ctrl);
  if (e.altKey) mods.push(MOD_NAMES.alt);
  if (e.shiftKey) mods.push(MOD_NAMES.shift);
  return { key: codeToKey(e.code), mods };
};

interface ShortcutInputProps {
  value: string;
  onChange: (v: string) => void;
  /**
   * 仅录制修饰键（用于 QuickPaste.modifier）。
   */
  modifierOnly?: boolean;
  placeholder?: string;
}

const ShortcutInput = ({
  value,
  onChange,
  modifierOnly = false,
  placeholder,
}: ShortcutInputProps) => {
  const { t } = useTranslation();
  const placeholderText = placeholder ?? t("shortcutInput.placeholder");
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<Captured>(EMPTY);
  const captureRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (open) captureRef.current?.focus();
  }, [open]);

  const reset = () => setDraft(EMPTY);

  const commit = (next: string) => {
    onChange(toCanonical(next));
    setOpen(false);
    reset();
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLElement>) => {
    e.preventDefault();
    e.stopPropagation();
    const cap = captureFromEvent(e);
    setDraft(cap);
    // 完整组合（修饰 + 主键）：keydown 即提交；纯修饰模式：等 Save 按钮，因为 keydown 阶段
    // 修饰键自身的事件 metaKey/ctrlKey 等已置位，可立刻预览，但 keyup 会清空 mods，不能在 keyup 提交。
    if (!modifierOnly && cap.key && cap.mods.length > 0) {
      commit(buildBinding(cap, false));
    }
  };

  const handleOpen = (next: boolean) => {
    setOpen(next);
    if (!next) reset();
  };

  const preview = buildBinding(draft, modifierOnly);

  return (
    <Popover isOpen={open} onOpenChange={handleOpen}>
      <Popover.Trigger>
        <button
          className={cn(
            "flex h-9 w-44 items-center rounded-md border border-default-200 bg-default-50 px-3 text-sm hover:bg-default-100",
            { "text-default-400": !value },
          )}
          type="button"
        >
          {value || placeholderText}
        </button>
      </Popover.Trigger>
      <Popover.Content>
        <Popover.Dialog className="w-64 p-3">
          <div className="mb-2 text-default-600 text-xs">
            {modifierOnly
              ? t("shortcutInput.hintModifierOnly")
              : t("shortcutInput.hint")}
          </div>
          <button
            className="flex h-12 w-full items-center justify-center rounded-md border border-default-300 border-dashed bg-default-50 text-sm outline-none focus:border-primary"
            onKeyDown={handleKeyDown}
            ref={captureRef}
            type="button"
          >
            {preview || (
              <span className="text-default-400">
                {t("shortcutInput.waiting")}
              </span>
            )}
          </button>
          <div className="mt-3 flex justify-between gap-2">
            <Button
              isDisabled={!value && draft.mods.length === 0}
              onPress={() => commit("")}
              size="sm"
              variant="ghost"
            >
              {t("shortcutInput.clear")}
            </Button>
            <div className="flex gap-2">
              <Button
                onPress={() => handleOpen(false)}
                size="sm"
                variant="ghost"
              >
                {t("shortcutInput.cancel")}
              </Button>
              {modifierOnly && (
                <Button
                  isDisabled={draft.mods.length === 0}
                  onPress={() => commit(buildBinding(draft, true))}
                  size="sm"
                >
                  {t("shortcutInput.save")}
                </Button>
              )}
            </div>
          </div>
        </Popover.Dialog>
      </Popover.Content>
    </Popover>
  );
};

export default ShortcutInput;
