import { invoke } from "@tauri-apps/api/core";
import { useDebounceFn } from "ahooks";
import { Button, Tooltip } from "antd";
import type { ChangeEvent, FC } from "react";
import KeyHint from "@/components/KeyHint";
import SearchInput from "@/components/SearchInput";
import { TAURI_COMMAND } from "@/constants/commands";
import { WINDOW_LABEL } from "@/constants/windows";
import { clipboardViewState } from "@/stores/clipboardView";
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

  /**
   * 防抖写入共享 store：连续打字时仅保留最后一次值，下游 List 直接消费 store 触发查询。
   * 搜索框自身不受 store 控制（非受控），避免 IME composition 期回灌导致重复字符。
   */
  const { run: handleKeywordChange } = useDebounceFn(
    (event: ChangeEvent<HTMLInputElement>) => {
      clipboardViewState.keyword = event.target.value.trim();
    },
    { wait: 200 },
  );

  return (
    <div
      className="flex items-center justify-between p-3"
      data-tauri-drag-region
    >
      <img alt="logo" className="size-5" src="/logo.png" />

      <div className="flex items-center gap-1">
        <SearchInput
          allowClear
          className="w-40"
          onChange={handleKeywordChange}
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

        <Tooltip title="更多操作">
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
