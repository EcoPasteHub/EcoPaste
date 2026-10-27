import { Input, type InputProps, type InputRef } from "antd";
import type { ChangeEvent, CompositionEvent, FC } from "react";
import { useCallback, useEffect, useRef } from "react";
import KeyHint from "@/components/KeyHint";
import { prepareClipboardWindowEditableFocus } from "@/hooks/useClipboardWindowEditableFocus";
import { useKeyboardEvent } from "@/hooks/useKeyboardEvent";

interface SearchInputProps extends Omit<InputProps, "prefix"> {
  blurToken?: number;
  clearToken?: number;
  focusToken?: number;
}

/**
 * 带快捷键提示的搜索输入框，支持 ⌘F / Ctrl+F 聚焦，未聚焦输入控件时直接打字也会聚焦。
 * IME 拼音/日文组合输入期间抑制 onChange，待 compositionend 再补发一次，
 * 避免上层防抖/受控逻辑被中间态拼字串污染。
 */
const SearchInput: FC<SearchInputProps> = (props) => {
  const {
    blurToken = 0,
    clearToken = 0,
    focusToken = 0,
    onChange,
    onCompositionStart,
    onCompositionEnd,
    ...rest
  } = props;

  const inputRef = useRef<InputRef>(null);
  const composingRef = useRef(false);

  /**
   * 聚焦搜索框并选中已有内容，便于直接覆盖输入。
   */
  const focusSearch = useCallback(async () => {
    if (!inputRef.current) return;

    await prepareClipboardWindowEditableFocus();
    inputRef.current?.focus({ cursor: "all" });
  }, []);

  /**
   * 未聚焦任何输入控件时敲下可打印字符，直接把焦点交给搜索框（type-to-search）。
   * 不阻止默认行为：keydown 期间切换焦点后，本次按键产生的字符会落入搜索框。
   */
  const handleTypeToSearch = (event: KeyboardEvent) => {
    if (!inputRef.current?.input) return;
    if (!isTypeToSearchKey(event)) return;
    if (isEditableElement(document.activeElement)) return;

    inputRef.current.focus({ cursor: "end" });
  };

  useKeyboardEvent("keydown", handleTypeToSearch);

  useEffect(() => {
    if (blurToken <= 0) return;

    inputRef.current?.blur();
  }, [blurToken]);

  useEffect(() => {
    if (focusToken <= 0) return;

    const frame = requestAnimationFrame(() => {
      void focusSearch();
    });

    return () => {
      cancelAnimationFrame(frame);
    };
  }, [focusToken, focusSearch]);

  const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    if (composingRef.current) return;

    onChange?.(event);
  };

  const handleCompositionStart = (
    event: CompositionEvent<HTMLInputElement>,
  ) => {
    composingRef.current = true;

    onCompositionStart?.(event);
  };

  const handleCompositionEnd = (event: CompositionEvent<HTMLInputElement>) => {
    composingRef.current = false;

    onCompositionEnd?.(event);
    // composition 结束时浏览器已派发最后一次 input，但被上面挡掉了，这里补一次。
    onChange?.(event as unknown as ChangeEvent<HTMLInputElement>);
  };

  return (
    <Input
      autoCapitalize="off"
      autoCorrect="off"
      data-allow-global-keyboard="true"
      key={clearToken}
      onChange={handleChange}
      onCompositionEnd={handleCompositionEnd}
      onCompositionStart={handleCompositionStart}
      prefix={
        <KeyHint
          hintKey="F"
          iconName="i-lucide:search"
          onKeyPress={focusSearch}
        />
      }
      ref={inputRef}
      spellCheck={false}
      {...rest}
    />
  );
};

/**
 * 是否为可触发 type-to-search 的按键：单个可打印字符且未按住修饰键。
 * 空格保留给列表预览，IME 组合中的按键交给输入法处理。
 */
function isTypeToSearchKey(event: KeyboardEvent) {
  if (event.defaultPrevented || event.isComposing) return false;
  if (event.metaKey || event.ctrlKey || event.altKey) return false;
  if (event.key.length !== 1 || event.key === " ") return false;

  return true;
}

function isEditableElement(element: Element | null) {
  if (!(element instanceof HTMLElement)) return false;
  if (element.isContentEditable) return true;

  return element.matches("input, textarea, select");
}

export default SearchInput;
