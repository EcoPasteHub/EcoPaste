import { Divider, Space, Switch } from "antd";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import { playCopySound } from "@/commands";
import Tooltip from "@/components/Tooltip";
import { cn } from "@/utils/cn";
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
  const { disabled = false, onChange, setting, value } = props;
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
            <Divider orientation="vertical" />
            <Tooltip title={t("schema.settings.copy.sound.preview")}>
              <button
                aria-label={t("schema.settings.copy.sound.preview")}
                className={cn(
                  "flex cursor-pointer appearance-none items-center justify-center border-0 bg-transparent p-0 text-ant-tertiary text-xl transition-colors hover:text-ant-primary focus-visible:text-ant-primary motion-reduce:transition-none",
                  {
                    "cursor-not-allowed text-ant-disabled hover:text-ant-disabled focus-visible:text-ant-disabled":
                      disabled,
                  },
                )}
                disabled={disabled}
                onClick={handlePreviewSound}
                type="button"
              >
                <i aria-hidden="true" className="i-lucide:play" />
              </button>
            </Tooltip>
          </>
        ) : null}
      </Space>
    </ControlFrame>
  );
};

export default SwitchControl;
