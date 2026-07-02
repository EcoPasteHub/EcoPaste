import { useDebounceFn } from "ahooks";
import type { MenuProps } from "antd";
import type { ChangeEvent, FC } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import {
  clearClipboardItems,
  setClipboardWindowPinned,
  showWindow,
} from "@/commands";
import CustomIconButton from "@/components/CustomIconButton";
import Dropdown, {
  type AppDropdownProps,
  type DropdownMenuItems,
} from "@/components/Dropdown";
import KeyHint from "@/components/KeyHint";
import Popover from "@/components/Popover";
import Tooltip from "@/components/Tooltip";
import { TAURI_EVENT } from "@/constants/events";
import { WINDOW_LABEL } from "@/constants/windows";
import { useTauriListen } from "@/hooks/useTauriListen";
import { clipboardStatsState } from "@/stores/clipboardStats";
import { clipboardViewState } from "@/stores/clipboardView";
import { settingsState } from "@/stores/settings";
import { cn } from "@/utils/cn";
import { formatShortcutDisplay } from "@/utils/shortcut";
import { isClipboardBottomSheet, usesClipboardSheetLayout } from "../layout";
import SearchInput from "./SearchInput";
import ShortcutList from "./ShortcutList";

interface WindowVisibilityPayload {
  label: string;
  visible: boolean;
}

type HeaderMoreMenuKey = "clear" | "preference";

const MORE_ACTION_TRIGGER: AppDropdownProps["trigger"] = ["click"];
const PREFERENCE_SHORTCUT = formatShortcutDisplay("CmdOrCtrl+,", " ");

/**
 * 剪贴板窗口顶部条：logo、搜索框（⌘F / Ctrl+F 聚焦）、固定窗口与更多操作入口。
 */
const Header: FC = () => {
  const { t } = useTranslation("clipboard");
  const settings = useSnapshot(settingsState);
  const { total } = useSnapshot(clipboardStatsState);
  const [pinned, setPinned] = useState(false);
  const [shortcutPopoverOpen, setShortcutPopoverOpen] = useState(false);
  const [searchBlurToken, setSearchBlurToken] = useState(0);
  const [searchClearToken, setSearchClearToken] = useState(0);
  const [searchFocusToken, setSearchFocusToken] = useState(0);
  const windowPosition = settings.clipboard.window.position;
  const isBottomSheet = isClipboardBottomSheet(windowPosition);
  const isFloatingSheet =
    usesClipboardSheetLayout(windowPosition) && !isBottomSheet;
  const isSheetLayout = usesClipboardSheetLayout(windowPosition);

  /**
   * 统一处理偏好设置入口（按钮点击/快捷键）。
   */
  const handleOpenPreference = () => {
    return showWindow(WINDOW_LABEL.PREFERENCE);
  };

  /**
   * 清空剪贴板历史；确认、toast 与后端调用统一收口在命令包装内。
   */
  const handleClearClipboardItems = async () => {
    await clearClipboardItems();
  };

  /**
   * 更多操作菜单分发：危险操作走确认弹窗，偏好设置打开独立窗口。
   */
  const handleMoreMenuClick: MenuProps["onClick"] = async (info) => {
    const key = info.key as HeaderMoreMenuKey;

    if (key === "clear") {
      await handleClearClipboardItems();

      return;
    }

    await handleOpenPreference();
  };

  /**
   * 切换剪贴板窗口固定态：Rust 侧立即生效（resign_key / 外部点击钩子读取），本地态仅用于按钮渲染。
   */
  const handleTogglePinned = async () => {
    const next = !pinned;

    await setClipboardWindowPinned(next);
    setPinned(next);
  };

  const handleShortcutKeyPress = () => {
    setShortcutPopoverOpen((current) => {
      return !current;
    });
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
   * 剪贴板窗口显隐变化时执行搜索框偏好：下次显示时清空关键词，显示后按设置自动聚焦。
   */
  const handleWindowVisibility = (event: {
    payload: WindowVisibilityPayload;
  }) => {
    const { label, visible } = event.payload;
    if (label !== WINDOW_LABEL.CLIPBOARD) return;

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

  const moreMenuItems: DropdownMenuItems = [
    {
      extra: PREFERENCE_SHORTCUT,
      icon: "i-lucide:settings",
      key: "preference",
      label: t("header.openPreference"),
    },
    {
      danger: true,
      icon: "i-lucide:trash-2",
      key: "clear",
      label: t("header.clearRecords"),
    },
  ];

  return (
    <div
      className={cn("p-3 pb-2", {
        "flex items-center": !isFloatingSheet,
        "justify-between": !isSheetLayout,
        "relative flex flex-col gap-2 px-5 pt-3 pb-2": isFloatingSheet,
        "relative h-16 justify-center px-5 py-2.5": isBottomSheet,
      })}
      data-tauri-drag-region
    >
      <div
        className={cn("flex items-center", {
          "absolute left-5 gap-2": isBottomSheet,
          "gap-2": isFloatingSheet,
        })}
      >
        <img
          alt={t("header.logoAlt")}
          className={cn({
            "size-5": !isSheetLayout,
            "size-6": isSheetLayout,
          })}
          src="/logo.png"
        />

        {isSheetLayout ? (
          <>
            <span className="text-ant-tertiary text-xs">
              {t("footer.total", { count: total ?? 0 })}
            </span>

            <Popover
              content={<ShortcutList bottomSheet />}
              onOpenChange={setShortcutPopoverOpen}
              open={shortcutPopoverOpen}
              placement="bottomLeft"
              styles={{ content: { padding: 0 } }}
              title={t("footer.shortcuts")}
              tooltip={t("footer.shortcuts")}
              trigger="click"
            >
              <button
                className="flex h-8 cursor-pointer items-center gap-1 rounded-1.5 border-0 bg-transparent px-2 text-ant-text text-xs shadow-none outline-none transition-colors hover:bg-ant-fill-tertiary focus-visible:bg-ant-fill-tertiary motion-reduce:transition-none"
                tabIndex={-1}
                type="button"
              >
                <KeyHint
                  className="size-4"
                  hintKey="K"
                  iconName="i-lucide:keyboard text-base"
                  onKeyPress={handleShortcutKeyPress}
                />
                {t("footer.shortcuts")}
              </button>
            </Popover>
          </>
        ) : null}
      </div>

      <div
        className={cn("flex items-center", {
          "w-96 max-w-[38vw]": isBottomSheet,
          "w-full": isFloatingSheet,
        })}
      >
        <SearchInput
          allowClear
          blurToken={searchBlurToken}
          className={cn({
            "w-40": !isSheetLayout,
            "w-96 max-w-[38vw]": isBottomSheet,
            "w-full": isFloatingSheet,
          })}
          clearToken={searchClearToken}
          focusToken={searchFocusToken}
          onChange={handleKeywordChange}
          placeholder={t("header.searchPlaceholder")}
          size={isSheetLayout ? "large" : "small"}
        />
      </div>

      <div
        className={cn("flex items-center gap-1", {
          "absolute right-5 gap-1.5": isBottomSheet,
          "absolute top-3 right-5 gap-1.5": isFloatingSheet,
        })}
      >
        <Tooltip title={t(pinned ? "header.unpin" : "header.pin")}>
          <CustomIconButton
            className={cn({
              "h-8 px-2 text-xs": isSheetLayout,
            })}
            icon={
              <KeyHint
                className={cn({ "size-4": isSheetLayout })}
                hintKey="P"
                iconName={cn("i-lets-icons:pin", {
                  "text-base": isSheetLayout,
                })}
                onKeyPress={handleTogglePinned}
              />
            }
            iconClassName={cn({ "text-base": isSheetLayout })}
            onClick={handleTogglePinned}
            size="small"
            type={pinned ? "primary" : "text"}
          >
            {isSheetLayout ? t(pinned ? "header.unpin" : "header.pin") : null}
          </CustomIconButton>
        </Tooltip>

        <Dropdown
          menu={{ items: moreMenuItems, onClick: handleMoreMenuClick }}
          tooltip={t("header.moreActions")}
          trigger={MORE_ACTION_TRIGGER}
        >
          <CustomIconButton
            className={cn({
              "h-8 px-2 text-xs": isSheetLayout,
            })}
            icon={
              <KeyHint
                className={cn({ "size-4": isSheetLayout })}
                hintKey=","
                iconName={cn("i-lets-icons:meatballs-menu", {
                  "text-base": isSheetLayout,
                })}
                onKeyPress={handleOpenPreference}
              />
            }
            iconClassName={cn({ "text-base": isSheetLayout })}
            size="small"
            type="text"
          >
            {isSheetLayout ? t("header.moreActions") : null}
          </CustomIconButton>
        </Dropdown>
      </div>
    </div>
  );
};

export default Header;
