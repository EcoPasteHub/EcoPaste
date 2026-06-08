import { Input, type InputProps, type InputRef } from "antd";
import type { ChangeEvent, CompositionEvent, FC } from "react";
import { useEffect, useRef } from "react";
import KeyHint from "@/components/KeyHint";
import { useMainWindowTextInputFocus } from "@/hooks/useMainWindowTextInputFocus";

interface SearchInputProps extends Omit<InputProps, "prefix"> {
  blurToken?: number;
  clearToken?: number;
  focusToken?: number;
}

/**
 * 带快捷键提示的搜索输入框，支持 ⌘F / Ctrl+F 聚焦。
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
    onBlur,
    onFocus,
    ...rest
  } = props;

  const inputRef = useRef<InputRef>(null);
  const composingRef = useRef(false);
  const inputFocusHandlers = useMainWindowTextInputFocus<HTMLInputElement>({
    onBlur,
    onFocus,
  });
  const {
    focusWindowForTextInput,
    onBlur: handleInputBlur,
    onFocus: handleInputFocus,
  } = inputFocusHandlers;

  /**
   * 聚焦搜索框并选中已有内容，便于直接覆盖输入。
   */
  const focusSearch = async () => {
    await focusWindowForTextInput();

    inputRef.current?.focus({ cursor: "all" });
  };

  useEffect(() => {
    if (blurToken <= 0) return;

    inputRef.current?.blur();
  }, [blurToken]);

  useEffect(() => {
    if (focusToken <= 0) return;

    const frame = requestAnimationFrame(async () => {
      await focusWindowForTextInput();

      inputRef.current?.focus({ cursor: "all" });
    });

    return () => {
      cancelAnimationFrame(frame);
    };
  }, [focusToken, focusWindowForTextInput]);

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
      key={clearToken}
      onBlur={handleInputBlur}
      onChange={handleChange}
      onCompositionEnd={handleCompositionEnd}
      onCompositionStart={handleCompositionStart}
      onFocus={handleInputFocus}
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

export default SearchInput;
