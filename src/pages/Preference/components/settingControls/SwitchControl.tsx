import { Button, Divider, Space, Switch } from "antd";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import { playCopySound } from "@/commands";
import Tooltip from "@/components/Tooltip";
import type { PreferenceSetting } from "../../types/preferences";
import ControlFrame from "./ControlFrame";
import type { ControlProps } from "./types";

const COPY_SOUND_SETTING_ID = "copy.sound";

interface SwitchControlProps extends ControlProps {
  setting: PreferenceSetting;
  value: boolean;
}

/**
 * 即时保存二元设置。
 */
const SwitchControl: FC<SwitchControlProps> = (props) => {
  const { t } = useTranslation("preferences");
  const { disabled, onChange, setting, value } = props;
  const canPreviewSound = setting.id === COPY_SOUND_SETTING_ID;

  const handleChange = async (checked: boolean) => {
    await onChange(setting, checked);
  };

  const handlePreviewSound = async () => {
    await playCopySound();
  };

  return (
    <ControlFrame>
      <Space size={0}>
        <Switch checked={value} disabled={disabled} onChange={handleChange} />
        {canPreviewSound ? (
          <>
            <Divider type="vertical" />
            <Tooltip title={t("schema.settings.copy.sound.preview")}>
              <Button
                aria-label={t("schema.settings.copy.sound.preview")}
                disabled={disabled}
                htmlType="button"
                icon={<i aria-hidden="true" className="i-lucide:play" />}
                onClick={handlePreviewSound}
                type="text"
              />
            </Tooltip>
          </>
        ) : null}
      </Space>
    </ControlFrame>
  );
};

export default SwitchControl;
