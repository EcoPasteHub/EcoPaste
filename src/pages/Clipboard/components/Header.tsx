import { useDebounceFn } from "ahooks";
import { Button } from "antd";
import type { ChangeEvent, FC } from "react";
import { useState } from "react";
import { setMainWindowPinned, showWindow } from "@/commands";
import KeyHint from "@/components/KeyHint";
import SearchInput from "@/components/SearchInput";
import Tooltip from "@/components/Tooltip";
import { WINDOW_LABEL } from "@/constants/windows";
import { clipboardViewState } from "@/stores/clipboardView";

/**
 * 剪贴板窗口顶部条：logo、搜索框（⌘F / Ctrl+F 聚焦）、固定窗口与更多操作入口。
 */
const Header: FC = () => {
  const [pinned, setPinned] = useState(false);

  /**
   * 统一处理偏好设置入口（按钮点击/快捷键）。
   */
  const handleOpenPreference = () => showWindow(WINDOW_LABEL.PREFERENCE);

  /**
   * 切换主窗口固定态：Rust 侧立即生效（resign_key / 外部点击钩子读取），本地态仅用于按钮渲染。
   */
  const handleTogglePinned = async () => {
    const next = !pinned;

    await setMainWindowPinned(next);
    setPinned(next);
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

        <Tooltip title={pinned ? "取消固定" : "固定窗口"}>
          <Button
            icon={
              <KeyHint
                hintKey="P"
                iconName="i-lets-icons:pin"
                onKeyPress={handleTogglePinned}
              />
            }
            onClick={handleTogglePinned}
            size="small"
            type={pinned ? "primary" : "text"}
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
