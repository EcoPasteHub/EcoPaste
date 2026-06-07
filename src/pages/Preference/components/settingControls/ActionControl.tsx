import { open } from "@tauri-apps/plugin-dialog";
import { Button, Modal, Space, Tooltip } from "antd";
import type { FC } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type ChangeStorageLocationResult,
  type CleanCacheResult,
  changeStorageLocation,
  cleanResourceCache,
  type ExportHistoryBackupResult,
  inspectHistoryBackup,
  openPreferenceDirectory,
  resetStorageLocation,
  type StorageLocation,
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
    result?:
      | ChangeStorageLocationResult
      | CleanCacheResult
      | ExportHistoryBackupResult,
  ) => void;
  storageLocation: StorageLocation | null;
}

/**
 * 展示右侧操作按钮，所有 action 控件保持同一尺寸。
 */
const ActionControl: FC<ActionControlProps> = (props) => {
  const { t } = useTranslation(["preferences", "common"]);
  const { disabled, setting, storageLocation, onActionComplete } = props;
  const [loading, setLoading] = useState(false);
  const [exportModalOpen, setExportModalOpen] = useState(false);

  if (setting.control.type !== "action") return null;

  const markActionComplete = (
    result?:
      | ChangeStorageLocationResult
      | CleanCacheResult
      | ExportHistoryBackupResult,
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

  const changeDataDirectory = async () => {
    setLoading(true);
    try {
      const selected = await open({
        directory: true,
        multiple: false,
        title: t("preferences:storageLocation.pickTitle"),
      });
      if (!selected || Array.isArray(selected)) return;

      const result = await changeStorageLocation(selected);
      markActionComplete(result);
    } finally {
      setLoading(false);
    }
  };

  const resetDataDirectory = async () => {
    setLoading(true);
    try {
      const result = await resetStorageLocation();
      markActionComplete(result);
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

  const confirmResetDataDirectory = () => {
    Modal.confirm({
      cancelText: t("common:actions.cancel"),
      content: t("preferences:storageLocation.resetConfirmContent"),
      okText: t("preferences:storageLocation.reset"),
      onOk: resetDataDirectory,
      title: t("preferences:storageLocation.resetConfirmTitle"),
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

  const openDataDirectory = async () => {
    setLoading(true);
    try {
      await openPreferenceDirectory("data");
      markActionComplete();
    } finally {
      setLoading(false);
    }
  };

  if (setting.id === DATA_DIRECTORY_SETTING_ID) {
    const canReset = storageLocation?.isCustom === true;

    return (
      <ControlFrame>
        <Space.Compact>
          <Tooltip title={t("preferences:storageLocation.open")}>
            <Button
              disabled={disabled || loading}
              icon={<i aria-hidden="true" className="i-lucide:folder-open" />}
              loading={loading}
              onClick={openDataDirectory}
              type="default"
            />
          </Tooltip>

          <Tooltip title={t("preferences:storageLocation.change")}>
            <Button
              disabled={disabled || loading}
              icon={<i aria-hidden="true" className="i-lucide:folder-sync" />}
              loading={loading}
              onClick={changeDataDirectory}
              type="default"
            />
          </Tooltip>

          <Tooltip title={t("preferences:storageLocation.reset")}>
            <Button
              disabled={disabled || loading || !canReset}
              icon={<i aria-hidden="true" className="i-lucide:rotate-ccw" />}
              loading={loading}
              onClick={confirmResetDataDirectory}
              type="default"
            />
          </Tooltip>
        </Space.Compact>
      </ControlFrame>
    );
  }

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
