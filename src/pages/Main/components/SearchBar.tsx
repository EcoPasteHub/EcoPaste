import { SearchField } from "@heroui/react";
import type { Ref } from "react";

interface Props {
  value: string;
  onChange: (next: string) => void;
  inputRef?: Ref<HTMLInputElement>;
}

const SearchBar = ({ value, onChange, inputRef }: Props) => {
  return (
    <SearchField
      aria-label="搜索剪贴板"
      className="px-2 py-1.5"
      onChange={onChange}
      value={value}
    >
      <SearchField.Group>
        <SearchField.SearchIcon />
        <SearchField.Input placeholder="搜索..." ref={inputRef} />
        <SearchField.ClearButton />
      </SearchField.Group>
    </SearchField>
  );
};

export default SearchBar;
