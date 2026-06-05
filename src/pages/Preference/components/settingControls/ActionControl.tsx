import { Button } from "antd";
import type { FC } from "react";
import { openPreferenceDirectory } from "@/commands";
import type { PreferenceSetting } from "../../types/preferences";
import ControlFrame from "./ControlFrame";

const DATA_DIRECTORY_SETTING_ID = "localData.dataDirectory";
const LOG_DIRECTORY_SETTING_ID = "localData.logDirectory";

interface ActionControlProps {
  disabled: boolean;
  setting: PreferenceSetting;
}

/**
 * 展示右侧操作按钮，所有 action 控件保持同一尺寸。
 */
const ActionControl: FC<ActionControlProps> = (props) => {
  const { disabled, setting } = props;

  if (setting.control.type !== "action") return null;

  const handleClick = async () => {
    if (setting.id === DATA_DIRECTORY_SETTING_ID) {
      await openPreferenceDirectory("data");
      return;
    }

    if (setting.id === LOG_DIRECTORY_SETTING_ID) {
      await openPreferenceDirectory("logs");
    }
  };

  return (
    <ControlFrame>
      <Button
        danger={setting.control.danger}
        disabled={disabled}
        onClick={handleClick}
        type="default"
      >
        {setting.control.label}
      </Button>
    </ControlFrame>
  );
};

export default ActionControl;
