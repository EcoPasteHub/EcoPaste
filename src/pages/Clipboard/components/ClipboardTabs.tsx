import { Tabs } from "@heroui/react";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";

import { TAURI_EVENT } from "@/constants/events";
import { useTauriListen } from "@/hooks/useTauriListen";
import {
  clipboardViewState,
  cycleTabKey,
  keyToTab,
  setClipboardTab,
  tabToKey,
} from "@/stores/clipboardView";

import { useClipboardGroups } from "../hooks/useClipboardGroups";

interface NavPayload {
  action: string;
}

/**
 * 横向 tab：固定「全部 / 收藏」+ 后续追加的分组。
 * 没有分组数据时仍渲染两个固定 tab，不退化为空——保持视觉位置稳定。
 * 键盘 Tab / Shift+Tab 在 tab 列表中循环切换；macOS 走 window keydown（与 useListNavigation 同款），
 * Windows 主窗 focusable=false 拿不到键，由 Rust 钩子 emit `keyboard://nav` 的 `nextTab`/`prevTab` 投递过来——两路同时挂，平台侧另一路自然不触发。
 */
const ClipboardTabs = () => {
  const { t } = useTranslation();
  const { tab } = useSnapshot(clipboardViewState);
  const groups = useClipboardGroups();

  // 完整 key 序列（含动态分组），用于循环切换。
  const orderedKeys = useMemo(
    () => ["all", "favorite", ...groups.map((g) => `group:${g.id}`)],
    [groups],
  );
  const keysRef = useRef(orderedKeys);
  keysRef.current = orderedKeys;

  const cycle = useCallback((direction: "next" | "prev") => {
    const current = tabToKey(clipboardViewState.tab);
    const next = cycleTabKey(keysRef.current, current, direction);
    if (next !== current) setClipboardTab(keyToTab(next));
  }, []);

  // macOS 路径：window keydown。capture + stopPropagation，避免 SearchField 把
  // Tab 当焦点切换吃掉。preventDefault 抑制浏览器默认 Tab 焦点行为。
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key !== "Tab") return;
      e.preventDefault();
      e.stopPropagation();
      cycle(e.shiftKey ? "prev" : "next");
    };
    window.addEventListener("keydown", handler, true);
    return () => window.removeEventListener("keydown", handler, true);
  }, [cycle]);

  // Windows 路径：复用 useListNavigation 用的同一个 NAV_EVENT 通道，
  // 只接 nextTab / prevTab 两个 action；其他 action 由 useListNavigation 处理。
  useTauriListen<NavPayload>(TAURI_EVENT.KEYBOARD_NAV, (payload) => {
    if (payload.action === "nextTab") cycle("next");
    else if (payload.action === "prevTab") cycle("prev");
  });

  return (
    <Tabs
      aria-label={t("clipboard.tabs.ariaLabel")}
      className="px-2"
      onSelectionChange={(key) => setClipboardTab(keyToTab(String(key)))}
      selectedKey={tabToKey(tab)}
    >
      <Tabs.List className="gap-1">
        <Tabs.Tab id="all">
          {t("clipboard.tabs.all")}
          <Tabs.Indicator />
        </Tabs.Tab>
        <Tabs.Tab id="favorite">
          {t("clipboard.tabs.favorite")}
          <Tabs.Indicator />
        </Tabs.Tab>
        {groups.map((group) => (
          <Tabs.Tab id={`group:${group.id}`} key={group.id}>
            {group.name}
            <Tabs.Indicator />
          </Tabs.Tab>
        ))}
      </Tabs.List>
    </Tabs>
  );
};

export default ClipboardTabs;
