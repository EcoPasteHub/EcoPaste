import { Input, type InputProps, type InputRef } from "antd";
import type { FC } from "react";
import { useRef } from "react";
import KeyHint from "@/components/KeyHint";

/**
 * 带快捷键提示的搜索输入框，支持 ⌘F / Ctrl+F 聚焦。
 * 内置搜索图标与 KeyHint，统一剪贴板与偏好设置的搜索交互。
 */
const SearchInput: FC<Omit<InputProps, "prefix">> = (props) => {
  const { ...restProps } = props;

  const inputRef = useRef<InputRef>(null);

  /**
   * 聚焦搜索框并选中已有内容，便于直接覆盖输入。
   */
  const focusSearch = () => {
    inputRef.current?.focus({ cursor: "all" });
  };

  return (
    <Input
      prefix={
        <KeyHint hintKey="F" onKeyPress={focusSearch}>
          <i className="i-lucide:search flex size-4" />
        </KeyHint>
      }
      ref={inputRef}
      {...restProps}
    />
  );
};

export default SearchInput;
