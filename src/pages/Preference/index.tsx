import { getName, getVersion } from "@tauri-apps/api/app";
import { useMount } from "ahooks";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import type { ChangeEvent, FC } from "react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import {
  type BackupReceivedPayload,
  type ChangeStorageLocationResult,
  type CleanCacheResult,
  type ExportHistoryBackupResult,
  getStorageLocation,
  getStorageUsage,
  type ImportHistoryBackupResult,
  type StorageLocation,
  type StorageUsage,
  takePendingBackup,
  takePendingPreferenceHighlight,
} from "@/commands";
import { TAURI_EVENT } from "@/constants/events";
import { useTauriListen } from "@/hooks/useTauriListen";
import { settingsState } from "@/stores/settings";
import { preloadSourceApps, reloadSourceApps } from "@/stores/sourceApps";
import type { Settings } from "@/types/settings";
import { cn } from "@/utils/cn";
import { log } from "@/utils/log";
import BackupImportModal from "./components/BackupImportModal";
import PreferenceAboutPanel from "./components/PreferenceAboutPanel";
import PreferenceHeader from "./components/PreferenceHeader";
import PreferenceSection from "./components/PreferenceSection";
import PreferenceSidebar from "./components/PreferenceSidebar";
import { preferenceTabs } from "./config/preferenceSchema";
import {
  commitSettingChange,
  settingValuesEqual,
} from "./services/preferenceSettings";
import type {
  PreferenceSetting,
  PreferenceStorageState,
  PreferenceTabId,
  SettingValue,
} from "./types/preferences";
import {
  resetContentScroll,
  scrollHighlightedSetting,
} from "./utils/preferenceScroll";
import {
  type PreferenceSearchResult,
  searchPreferenceSettings,
} from "./utils/preferenceSearch";

type PreferenceHighlightTarget = {
  settingId: string;
  token: number;
};

interface PreferenceHighlightSettingPayload {
  settingId: string;
}

type AppMetadata = {
  name: string;
  version: string;
};

/**
 * EcoPaste 偏好设置：以用户心智组织设置，而非代码模块。
 */
const Preference: FC = () => {
  const { t } = useTranslation("preferences");
  const settings = useSnapshot(settingsState) as Settings;
  const shouldReduceMotion = useReducedMotion();
  const reduceMotion = shouldReduceMotion === true;
  const contentRef = useRef<HTMLDivElement | null>(null);
  const [activeTabId, setActiveTabId] = useState<PreferenceTabId>("record");
  const [activeSectionId, setActiveSectionId] = useState("capture");
  const [searchQuery, setSearchQuery] = useState("");
  const [highlightTarget, setHighlightTarget] =
    useState<PreferenceHighlightTarget | null>(null);
  const [appMetadata, setAppMetadata] = useState<AppMetadata>({
    name: "",
    version: "",
  });
  const [storageState, setStorageState] =
    useState<PreferenceStorageState>("loading");
  const [storageUsage, setStorageUsage] = useState<StorageUsage | null>(null);
  const [storageLocation, setStorageLocation] =
    useState<StorageLocation | null>(null);
  const [backupImportTarget, setBackupImportTarget] =
    useState<BackupReceivedPayload | null>(null);

  const activeTab =
    preferenceTabs.find((tab) => {
      return tab.id === activeTabId;
    }) ?? preferenceTabs[0];
  const searchResults = useMemo(() => {
    return searchPreferenceSettings(searchQuery, t);
  }, [searchQuery, t]);
  const totalSettings = activeTab.sections.reduce((total, section) => {
    return total + section.settings.length;
  }, 0);
  const activeSection =
    activeTab.sections.find((section) => {
      return section.id === activeSectionId;
    }) ?? activeTab.sections[0];
  const isAboutTab = activeTabId === "about";
  const isSourceSection = activeSection?.id === "source";

  const handleSearchChange = (event: ChangeEvent<HTMLInputElement>) => {
    setSearchQuery(event.target.value);
  };

  const handleTabSelect = (nextTabId: PreferenceTabId) => {
    const nextTab =
      preferenceTabs.find((tab) => {
        return tab.id === nextTabId;
      }) ?? preferenceTabs[0];
    const nextSectionId = nextTab.sections[0]?.id ?? "";

    setActiveTabId(nextTabId);
    setActiveSectionId(nextSectionId);
    resetContentScroll(contentRef.current);
  };

  const handleSectionSelect = (sectionId: string) => {
    setActiveSectionId(sectionId);
    resetContentScroll(contentRef.current);
  };

  const handlePickSearchResult = (result: PreferenceSearchResult) => {
    setActiveTabId(result.tab.id);
    setActiveSectionId(result.section.id);
    setSearchQuery("");
    highlightSetting(result.setting.id);
  };

  /**
   * 切换到指定设置项所属分类，并触发滚动高亮。
   */
  const highlightSetting = (settingId: string) => {
    const target = findPreferenceSetting(settingId);
    if (!target) return;

    setActiveTabId(target.tab.id);
    setActiveSectionId(target.section.id);
    setSearchQuery("");
    setHighlightTarget((currentTarget) => {
      return {
        settingId,
        token: (currentTarget?.token ?? 0) + 1,
      };
    });
  };

  const handleSettingChange = async (
    setting: PreferenceSetting,
    value: SettingValue,
  ) => {
    if (!setting.path) return;

    const currentValue = setting.value?.(settings);
    if (currentValue === void 0) return;
    if (settingValuesEqual(currentValue, value)) return;

    try {
      await commitSettingChange(setting, value);
    } catch {
      // 错误 toast 已由 commands 层统一处理；设置镜像等待 Rust 事件回灌。
    }
  };

  const highlightBackupImport = () => {
    setActiveTabId("data");
    setActiveSectionId("backup");
    setHighlightTarget((currentTarget) => {
      return {
        settingId: "backup.importHistory",
        token: (currentTarget?.token ?? 0) + 1,
      };
    });
  };

  const handleBackupReceived = (payload: BackupReceivedPayload) => {
    highlightBackupImport();
    setBackupImportTarget(payload);
  };

  /**
   * 关闭备份导入弹窗并丢弃当前接收的文件路径。
   */
  const closeBackupImportModal = () => {
    setBackupImportTarget(null);
  };

  /**
   * 导入完成后刷新存储占用。
   */
  const handleBackupImported = (_result: ImportHistoryBackupResult) => {
    closeBackupImportModal();
    void initializeStorageUsage();
  };

  const handleActionComplete = (
    setting: PreferenceSetting,
    result?:
      | ChangeStorageLocationResult
      | CleanCacheResult
      | ExportHistoryBackupResult,
  ) => {
    if (result && "location" in result) {
      setStorageLocation(result.location);
      setStorageUsage(result.storageUsage);
      setStorageState("ready");
      return;
    }

    if (result && "storageUsage" in result) {
      setStorageUsage(result.storageUsage);
      setStorageState("ready");
      return;
    }

    if (
      setting.id.startsWith("localData.") ||
      setting.id.startsWith("backup.")
    ) {
      void initializeStorageUsage();
    }
  };

  /**
   * 加载当前环境数据目录的递归占用，用于侧栏低频状态展示。
   */
  const initializeStorageUsage = async () => {
    setStorageState("loading");

    try {
      const [usage, location] = await Promise.all([
        getStorageUsage(),
        getStorageLocation(),
      ]);

      setStorageUsage(usage);
      setStorageLocation(location);
      setStorageState("ready");
    } catch (error) {
      log.warn("load storage usage failed", error);
      setStorageState("error");
    }
  };

  /**
   * 从 Tauri 应用元信息读取展示名称和版本，避免前端手写包信息。
   */
  const initializeAppMetadata = async () => {
    try {
      const [name, version] = await Promise.all([getName(), getVersion()]);

      setAppMetadata({ name, version });
    } catch (error) {
      log.warn("load app metadata failed", error);
    }
  };

  useMount(async () => {
    void initializeStorageUsage();
    void initializeAppMetadata();
    void preloadSourceApps();

    // 窗口空闲销毁后重建时，备份接收 / 定位高亮事件已无法 push 给刚挂载的前端，改为主动拉取暂存值。
    const pendingBackup = await takePendingBackup();

    if (pendingBackup) {
      handleBackupReceived(pendingBackup);
    }

    const pendingHighlight = await takePendingPreferenceHighlight();

    if (pendingHighlight) {
      highlightSetting(pendingHighlight);
    }
  });

  useTauriListen<BackupReceivedPayload>(
    TAURI_EVENT.BACKUP_RECEIVED,
    (event) => {
      handleBackupReceived(event.payload);
    },
  );

  useTauriListen<PreferenceHighlightSettingPayload>(
    TAURI_EVENT.PREFERENCE_HIGHLIGHT_SETTING,
    (event) => {
      highlightSetting(event.payload.settingId);
    },
  );

  useEffect(() => {
    if (!highlightTarget) return;

    const scrollTimer = window.setTimeout(
      () => {
        scrollHighlightedSetting(
          contentRef.current,
          highlightTarget.settingId,
          reduceMotion,
        );
      },
      reduceMotion ? 0 : 140,
    );
    const clearTimer = window.setTimeout(
      () => {
        setHighlightTarget((currentTarget) => {
          if (!currentTarget) return currentTarget;
          if (currentTarget.token !== highlightTarget.token) {
            return currentTarget;
          }

          return null;
        });
      },
      reduceMotion ? 1200 : 2200,
    );

    return () => {
      window.clearTimeout(scrollTimer);
      window.clearTimeout(clearTimer);
    };
  }, [highlightTarget, reduceMotion]);

  useEffect(() => {
    if (!isSourceSection) return;

    void reloadSourceApps();
  }, [isSourceSection]);

  if (!activeTab) return null;

  return (
    <div className="h-screen overflow-hidden bg-ant-layout text-ant-text">
      <motion.div
        animate={{ opacity: 1, y: 0 }}
        className="flex h-full overflow-hidden bg-ant-layout"
        data-tauri-drag-region
        initial={{ opacity: 0, y: reduceMotion ? 0 : 6 }}
        transition={{ duration: reduceMotion ? 0 : 0.18, ease: "easeOut" }}
      >
        <PreferenceSidebar
          activeTabId={activeTabId}
          appName={appMetadata.name}
          appVersion={appMetadata.version}
          onTabSelect={handleTabSelect}
          storageState={storageState}
          storageUsage={storageUsage}
        />

        <main className="flex min-w-0 flex-1 flex-col overflow-hidden bg-ant-layout">
          <PreferenceHeader
            activeSectionId={activeSectionId}
            activeTab={activeTab}
            isAboutTab={isAboutTab}
            onPickSearchResult={handlePickSearchResult}
            onSearchChange={handleSearchChange}
            onSectionSelect={handleSectionSelect}
            searchQuery={searchQuery}
            searchResults={searchResults}
            shouldReduceMotion={reduceMotion}
            totalSettings={totalSettings}
          />

          <div
            className="min-h-0 flex-1 overflow-auto p-6"
            data-tauri-drag-region
            ref={contentRef}
          >
            <AnimatePresence mode="wait">
              {isAboutTab ? (
                <motion.div
                  animate={{ opacity: 1 }}
                  className="flex max-w-252 flex-col"
                  exit={{ opacity: 0 }}
                  initial={{ opacity: 0 }}
                  key="about"
                  transition={{
                    duration: reduceMotion ? 0 : 0.12,
                    ease: "easeOut",
                  }}
                >
                  <PreferenceAboutPanel
                    appName={appMetadata.name}
                    appVersion={appMetadata.version}
                  />
                </motion.div>
              ) : activeSection ? (
                <motion.div
                  animate={{ opacity: 1 }}
                  className={cn(
                    "flex flex-col",
                    isSourceSection ? "h-full max-w-none" : "max-w-228",
                  )}
                  exit={{ opacity: 0 }}
                  initial={{ opacity: 0 }}
                  key={`${activeTabId}-${activeSection.id}`}
                  transition={{
                    duration: reduceMotion ? 0 : 0.12,
                    ease: "easeOut",
                  }}
                >
                  <PreferenceSection
                    highlightedSettingId={highlightTarget?.settingId ?? null}
                    highlightToken={highlightTarget?.token ?? 0}
                    onActionComplete={handleActionComplete}
                    onChange={handleSettingChange}
                    section={activeSection}
                    settings={settings}
                    shouldReduceMotion={reduceMotion}
                    storageLocation={storageLocation}
                  />
                </motion.div>
              ) : null}
            </AnimatePresence>
          </div>
        </main>
      </motion.div>

      <BackupImportModal
        onCancel={closeBackupImportModal}
        onImported={handleBackupImported}
        open={backupImportTarget !== null}
        target={backupImportTarget}
      />
    </div>
  );
};

/**
 * 从完整偏好设置 schema 中找到指定设置项及其所属层级。
 */
function findPreferenceSetting(settingId: string) {
  for (const tab of preferenceTabs) {
    for (const section of tab.sections) {
      const setting = section.settings.find((item) => {
        return item.id === settingId;
      });
      if (!setting) continue;

      return { section, setting, tab };
    }
  }

  return null;
}

export default Preference;
