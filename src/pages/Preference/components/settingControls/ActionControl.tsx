import { Button, Modal } from "antd";
import type { FC } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { openPreferenceDirectory } from "@/commands";
import { resetSettings } from "@/stores/settings";
import type { PreferenceSetting } from "../../types/preferences";
import { translatePreferenceControlLabel } from "../../utils/preferenceI18n";
import ControlFrame from "./ControlFrame";

const DATA_DIRECTORY_SETTING_ID = "localData.dataDirectory";
const LOG_DIRECTORY_SETTING_ID = "localData.logDirectory";
const RESET_PREFERENCES_SETTING_ID = "diagnostics.resetPreferences";

interface ActionControlProps {
  disabled: boolean;
  setting: PreferenceSetting;
}

/**
 * 展示右侧操作按钮，所有 action 控件保持同一尺寸。
 */
const ActionControl: FC<ActionControlProps> = (props) => {
  const { t } = useTranslation(["preferences", "common"]);
  const { disabled, setting } = props;
  const [loading, setLoading] = useState(false);

  if (setting.control.type !== "action") return null;

  const resetPreferenceSettings = async () => {
    setLoading(true);
    try {
      await resetSettings();
    } finally {
      setLoading(false);
    }
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
      return;
    }

    if (setting.id === LOG_DIRECTORY_SETTING_ID) {
      await openPreferenceDirectory("logs");
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
