import type { FC } from "react";
import { useTranslation } from "react-i18next";
import type { StorageUsage } from "@/commands";
import { cn } from "@/utils/cn";
import { preferenceTabs } from "../config/preferenceSchema";
import { APP_NAME_PLACEHOLDER, PREFERENCE_TAB_META } from "../constants";
import type {
  PreferenceStorageState,
  PreferenceTabId,
} from "../types/preferences";
import { translatePreferenceTab } from "../utils/preferenceI18n";
import PreferenceCountTag from "./PreferenceCountTag";
import PreferenceStorageUsagePanel from "./PreferenceStorageUsagePanel";

interface PreferenceSidebarProps {
  activeTabId: PreferenceTabId;
  appName: string;
  appVersion: string;
  storageState: PreferenceStorageState;
  storageUsage: StorageUsage | null;
  onTabSelect: (tabId: PreferenceTabId) => void;
}

/**
 * 偏好窗口左侧导航栏：展示应用身份、一级分类和本地存储概览。
 */
const PreferenceSidebar: FC<PreferenceSidebarProps> = (props) => {
  const { t } = useTranslation("preferences");
  const {
    activeTabId,
    appName,
    appVersion,
    storageState,
    storageUsage,
    onTabSelect,
  } = props;
  const appNameLabel = appName.length > 0 ? appName : APP_NAME_PLACEHOLDER;
  const appVersionLabel = appVersion.length > 0 ? `v${appVersion}` : "";

  return (
    <aside
      className="flex w-56 shrink-0 flex-col border-ant-border-secondary border-r bg-ant-container"
      data-tauri-drag-region
    >
      <div className="flex items-center gap-2.5 px-4 pt-10 pb-4">
        <img
          alt=""
          className="size-10 shrink-0 object-contain"
          draggable={false}
          src="/logo.png"
        />
        <div className="min-w-0">
          <div className="font-semibold text-ant-text text-sm leading-none">
            {appNameLabel}
          </div>
          {appVersionLabel.length > 0 ? (
            <PreferenceCountTag className="mt-1.5 text-ant-tertiary">
              {appVersionLabel}
            </PreferenceCountTag>
          ) : null}
        </div>
      </div>

      <nav
        className="flex flex-1 flex-col gap-0.5 px-3 pb-3"
        data-tauri-drag-region
      >
        {preferenceTabs.map((tab) => {
          const meta = PREFERENCE_TAB_META[tab.id];
          const selected = tab.id === activeTabId;
          const handleClick = () => {
            onTabSelect(tab.id);
          };

          return (
            <button
              className={cn(
                "group relative flex h-10 w-full cursor-pointer items-center gap-2 rounded-1.75 border-0 bg-transparent px-2 text-left transition-colors focus-visible:ring-1 focus-visible:ring-ant-primary motion-reduce:transition-none",
                selected
                  ? meta.activeClass
                  : "text-ant-secondary hover:bg-ant-fill-tertiary hover:text-ant-text",
              )}
              key={tab.id}
              onClick={handleClick}
              type="button"
            >
              <span
                className={cn(
                  "flex size-7.5 shrink-0 items-center justify-center text-lg transition-colors motion-reduce:transition-none",
                  selected
                    ? "text-ant-primary"
                    : "text-ant-tertiary group-hover:text-ant-secondary",
                )}
              >
                <i aria-hidden="true" className={meta.icon} />
              </span>
              <span className="min-w-0 flex-1 truncate font-medium text-sm leading-tight">
                {translatePreferenceTab(t, tab)}
              </span>
              <span
                className={cn(
                  "h-5 w-0.75 rounded-full transition-colors motion-reduce:transition-none",
                  selected ? "bg-ant-primary" : "bg-transparent",
                )}
              />
            </button>
          );
        })}
      </nav>

      <PreferenceStorageUsagePanel
        state={storageState}
        storageUsage={storageUsage}
      />
    </aside>
  );
};

export default PreferenceSidebar;
