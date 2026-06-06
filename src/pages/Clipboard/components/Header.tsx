import { useDebounceFn } from "ahooks";
import { Button } from "antd";
import type { ChangeEvent, FC } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import { setMainWindowPinned, showWindow } from "@/commands";
import KeyHint from "@/components/KeyHint";
import Tooltip from "@/components/Tooltip";
import { TAURI_EVENT } from "@/constants/events";
import { WINDOW_LABEL } from "@/constants/windows";
import { useTauriListen } from "@/hooks/useTauriListen";
import { clipboardViewState } from "@/stores/clipboardView";
import { settingsState } from "@/stores/settings";
import SearchInput from "./SearchInput";

interface WindowVisibilityPayload {
  label: string;
  visible: boolean;
}

/**
 * 剪贴板窗口顶部条：logo、搜索框（⌘F / Ctrl+F 聚焦）、固定窗口与更多操作入口。
 */
const Header: FC = () => {
  const { t } = useTranslation("clipboard");
  const settings = useSnapshot(settingsState);
  const [pinned, setPinned] = useState(false);
  const [searchBlurToken, setSearchBlurToken] = useState(0);
  const [searchClearToken, setSearchClearToken] = useState(0);
  const [searchFocusToken, setSearchFocusToken] = useState(0);

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
  const { cancel: cancelKeywordChange, run: handleKeywordChange } =
    useDebounceFn(
      (event: ChangeEvent<HTMLInputElement>) => {
        clipboardViewState.keyword = event.target.value.trim();
      },
      { wait: 200 },
    );

  /**
   * 递增 token 触发搜索框清空，同时同步查询状态回到完整列表。
   */
  const clearSearch = () => {
    cancelKeywordChange();
    clipboardViewState.keyword = "";

    setSearchClearToken((current) => {
      return current + 1;
    });
  };

  /**
   * 递增 token 让搜索框失焦，避免窗口重新打开时保留上一次 activeElement。
   */
  const blurSearch = () => {
    setSearchBlurToken((current) => {
      return current + 1;
    });
  };

  /**
   * 递增 token 触发搜索框在窗口完成显示后的下一帧聚焦。
   */
  const focusSearch = () => {
    setSearchFocusToken((current) => {
      return current + 1;
    });
  };

  /**
   * 主窗口显隐变化时执行搜索框偏好：下次显示时清空关键词，显示后按设置自动聚焦。
   */
  const handleWindowVisibility = (event: {
    payload: WindowVisibilityPayload;
  }) => {
    const { label, visible } = event.payload;
    if (label !== WINDOW_LABEL.MAIN) return;

    if (!visible) {
      blurSearch();

      if (settings.clipboard.search.clearOnHide) {
        clearSearch();
      }

      return;
    }

    if (settings.clipboard.search.clearOnHide) {
      clearSearch();
    }

    if (!settings.clipboard.search.defaultFocus) {
      blurSearch();

      return;
    }

    focusSearch();
  };

  useTauriListen<WindowVisibilityPayload>(
    TAURI_EVENT.WINDOW_VISIBILITY,
    handleWindowVisibility,
  );

  return (
    <div
      className="flex items-center justify-between p-3"
      data-tauri-drag-region
    >
      <img alt={t("header.logoAlt")} className="size-5" src="/logo.png" />

      <div className="flex items-center gap-1">
        <SearchInput
          allowClear
          blurToken={searchBlurToken}
          className="w-40"
          clearToken={searchClearToken}
          focusToken={searchFocusToken}
          onChange={handleKeywordChange}
          placeholder={t("header.searchPlaceholder")}
          size="small"
        />

        <Tooltip title={t(pinned ? "header.unpin" : "header.pin")}>
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

        <Tooltip title={t("header.moreActions")}>
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
