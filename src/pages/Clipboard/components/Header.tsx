import { Button, Tooltip } from "antd";
import KeyHint from "@/components/KeyHint";
import SearchInput from "@/components/SearchInput";

/**
 * 剪贴板窗口顶部条：logo、搜索框（⌘F / Ctrl+F 聚焦）、固定窗口与更多操作入口。
 */
const Header = () => {
  return (
    <div className="flex items-center justify-between" data-tauri-drag-region>
      <img alt="logo" className="size-5" src="/logo.png" />

      <div className="flex items-center gap-1">
        <SearchInput
          className="w-40"
          placeholder="搜索剪贴板..."
          size="small"
        />

        <Tooltip title="固定窗口">
          <Button
            icon={
              <KeyHint hintKey="P">
                <i className="i-lets-icons:pin text-4" />
              </KeyHint>
            }
            size="small"
            type="text"
          />
        </Tooltip>

        <Tooltip title="更多操作">
          <Button
            icon={<i className="i-lets-icons:meatballs-menu text-4" />}
            size="small"
            type="text"
          />
        </Tooltip>
      </div>
    </div>
  );
};

export default Header;
