import { open } from "@tauri-apps/plugin-dialog";
import { Button, Modal } from "antd";
import type { FC } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type CleanCacheResult,
  cleanResourceCache,
  type ExportHistoryBackupResult,
  inspectHistoryBackup,
  openPreferenceDirectory,
} from "@/commands";
import { resetSettings } from "@/stores/settings";
import { log } from "@/utils/log";
import type { PreferenceSetting } from "../../types/preferences";
import { translatePreferenceControlLabel } from "../../utils/preferenceI18n";
import BackupExportModal from "../BackupExportModal";
import ControlFrame from "./ControlFrame";

const BACKUP_EXTENSION = "ecopastebak";
const CLEAN_CACHE_SETTING_ID = "localData.cleanCache";
const DATA_DIRECTORY_SETTING_ID = "localData.dataDirectory";
const EXPORT_BACKUP_SETTING_ID = "backup.exportHistory";
const IMPORT_BACKUP_SETTING_ID = "backup.importHistory";
const LOG_DIRECTORY_SETTING_ID = "localData.logDirectory";
const RESET_PREFERENCES_SETTING_ID = "diagnostics.resetPreferences";

interface ActionControlProps {
  disabled: boolean;
  setting: PreferenceSetting;
  onActionComplete?: (
    setting: PreferenceSetting,
    result?: CleanCacheResult | ExportHistoryBackupResult,
  ) => void;
}

/**
 * 展示右侧操作按钮，所有 action 控件保持同一尺寸。
 */
const ActionControl: FC<ActionControlProps> = (props) => {
  const { t } = useTranslation(["preferences", "common"]);
  const { disabled, setting, onActionComplete } = props;
  const [loading, setLoading] = useState(false);
  const [exportModalOpen, setExportModalOpen] = useState(false);

  if (setting.control.type !== "action") return null;

  const markActionComplete = (
    result?: CleanCacheResult | ExportHistoryBackupResult,
  ) => {
    onActionComplete?.(setting, result);
  };

  const cleanCache = async () => {
    setLoading(true);
    try {
      const result = await cleanResourceCache();
      markActionComplete(result);
    } finally {
      setLoading(false);
    }
  };

  const resetPreferenceSettings = async () => {
    setLoading(true);
    try {
      await resetSettings();
      markActionComplete();
    } finally {
      setLoading(false);
    }
  };

  const confirmCleanCache = () => {
    Modal.confirm({
      cancelText: t("common:actions.cancel"),
      content: t(
        "preferences:schema.settings.localData.cleanCache.confirmContent",
      ),
      okText: t(
        "preferences:schema.settings.localData.cleanCache.controlLabel",
      ),
      onOk: cleanCache,
      title: t("preferences:schema.settings.localData.cleanCache.confirmTitle"),
    });
  };

  const confirmResetPreferences = () => {
    Modal.confirm({
      cancelText: t("common:actions.cancel"),
      content: t(
        "preferences:schema.settings.diagnostics.resetPreferences.confirmContent",
      ),
      okButtonProps: { danger: true },
      okText: t("common:actions.reset"),
      onOk: resetPreferenceSettings,
      title: t(
        "preferences:schema.settings.diagnostics.resetPreferences.confirmTitle",
      ),
    });
  };

  const openExportModal = () => {
    setExportModalOpen(true);
  };

  /**
   * 选择 `.ecopastebak` 文件并交给 Rust 识别，识别事件会打开统一导入弹窗。
   */
  const pickImportBackup = async () => {
    setLoading(true);
    try {
      const selected = await open({
        directory: false,
        filters: [
          {
            extensions: [BACKUP_EXTENSION],
            name: t("preferences:backup.fileFilter"),
          },
        ],
        multiple: false,
        title: t("preferences:backup.import.pickTitle"),
      });
      if (!selected || Array.isArray(selected)) return;

      await inspectHistoryBackup({
        path: selected,
        source: "openFile",
      });
    } catch (error) {
      log.warn("pick backup import file failed", error);
    } finally {
      setLoading(false);
    }
  };

  const closeExportModal = () => {
    setExportModalOpen(false);
  };

  const handleBackupExported = (result: ExportHistoryBackupResult) => {
    setExportModalOpen(false);
    markActionComplete(result);
  };

  const handleClick = async () => {
    if (setting.id === EXPORT_BACKUP_SETTING_ID) {
      openExportModal();
      return;
    }

    if (setting.id === IMPORT_BACKUP_SETTING_ID) {
      await pickImportBackup();
      return;
    }

    if (setting.id === DATA_DIRECTORY_SETTING_ID) {
      await openPreferenceDirectory("data");
      markActionComplete();
      return;
    }

    if (setting.id === LOG_DIRECTORY_SETTING_ID) {
      await openPreferenceDirectory("logs");
      markActionComplete();
      return;
    }

    if (setting.id === CLEAN_CACHE_SETTING_ID) {
      confirmCleanCache();
      return;
    }

    if (setting.id === RESET_PREFERENCES_SETTING_ID) {
      confirmResetPreferences();
    }
  };

  return (
    <>
      <ControlFrame>
        <Button
          color={setting.control.color ?? "primary"}
          disabled={disabled || loading}
          loading={loading}
          onClick={handleClick}
          variant="outlined"
        >
          {translatePreferenceControlLabel(t, setting)}
        </Button>
      </ControlFrame>

      {setting.id === EXPORT_BACKUP_SETTING_ID ? (
        <BackupExportModal
          onCancel={closeExportModal}
          onExported={handleBackupExported}
          open={exportModalOpen}
        />
      ) : null}
    </>
  );
};

export default ActionControl;
