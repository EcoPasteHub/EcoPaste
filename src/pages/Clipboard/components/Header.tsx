import { invoke } from "@tauri-apps/api/core";
import { Button, Tooltip } from "antd";
import type { FC } from "react";
import KeyHint from "@/components/KeyHint";
import SearchInput from "@/components/SearchInput";
import { TAURI_COMMAND } from "@/constants/commands";
import { WINDOW_LABEL } from "@/constants/windows";
import { log } from "@/utils/log";

/**
 * 剪贴板窗口顶部条：logo、搜索框（⌘F / Ctrl+F 聚焦）、固定窗口与更多操作入口。
 */
const Header: FC = () => {
  /**
   * 统一处理偏好设置入口（按钮点击/快捷键）。
   */
  const handleOpenPreference = async () => {
    try {
      await invoke<void>(TAURI_COMMAND.SHOW_WINDOW, {
        label: WINDOW_LABEL.PREFERENCE,
      });
    } catch (error) {
      log.error("Open preference window failed", error);
    }
  };

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
            icon={<KeyHint hintKey="P" iconName="i-lets-icons:pin" />}
            size="small"
            type="text"
          />
        </Tooltip>

        <Tooltip title="偏好设置">
          <Button
            icon={
              <KeyHint
                hintKey=","
                iconName="i-lets-icons:meatballs-menu"
                onKeyPress={handleOpenPreference}
              />
            }
            size="small"
            type="text"
          />
        </Tooltip>
      </div>
    </div>
  );
};

export default Header;
