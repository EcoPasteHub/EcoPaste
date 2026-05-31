import { Input, type InputRef } from "antd";
import type { Ref } from "react";

interface Props {
  value: string;
  onChange: (next: string) => void;
  // 接收 antd InputRef；调用方持有 InputRef，通过 .input 拿到原生 input DOM。
  inputRef?: Ref<InputRef>;
}

/**
 * 顶部搜索框：antd Input 受控；allowClear 提供清空按钮。
 */
const SearchBar = ({ value, onChange, inputRef }: Props) => {
  return (
    <div className="px-2 py-1.5">
      <Input
        allowClear
        aria-label="搜索剪贴板"
        onChange={(e) => onChange(e.target.value)}
        placeholder="搜索..."
        ref={inputRef}
        value={value}
      />
    </div>
  );
};

export default SearchBar;
