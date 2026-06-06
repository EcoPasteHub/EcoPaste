import { Button, Modal } from "antd";
import type { FC } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  type CleanCacheResult,
  cleanResourceCache,
  openPreferenceDirectory,
} from "@/commands";
import { resetSettings } from "@/stores/settings";
import type { PreferenceSetting } from "../../types/preferences";
import { translatePreferenceControlLabel } from "../../utils/preferenceI18n";
import ControlFrame from "./ControlFrame";

const CLEAN_CACHE_SETTING_ID = "localData.cleanCache";
const DATA_DIRECTORY_SETTING_ID = "localData.dataDirectory";
const LOG_DIRECTORY_SETTING_ID = "localData.logDirectory";
const RESET_PREFERENCES_SETTING_ID = "diagnostics.resetPreferences";

interface ActionControlProps {
  disabled: boolean;
  setting: PreferenceSetting;
  onActionComplete?: (
    setting: PreferenceSetting,
    result?: CleanCacheResult,
  ) => void;
}

/**
 * 展示右侧操作按钮，所有 action 控件保持同一尺寸。
 */
const ActionControl: FC<ActionControlProps> = (props) => {
  const { t } = useTranslation(["preferences", "common"]);
  const { disabled, setting, onActionComplete } = props;
  const [loading, setLoading] = useState(false);

  if (setting.control.type !== "action") return null;

  const markActionComplete = (result?: CleanCacheResult) => {
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

  const handleClick = async () => {
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
    <ControlFrame>
      <Button
        danger={setting.control.danger}
        disabled={disabled || loading}
        loading={loading}
        onClick={handleClick}
        type="default"
      >
        {translatePreferenceControlLabel(t, setting)}
      </Button>
    </ControlFrame>
  );
};

export default ActionControl;
