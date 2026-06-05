import { Divider, Switch } from "antd";
import type { FC } from "react";
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
  const { disabled, onChange, setting, value } = props;
  const canPreviewSound = setting.id === COPY_SOUND_SETTING_ID;

  const handleChange = async (checked: boolean) => {
    await onChange(setting, checked);
  };

  const handlePreviewSound = async () => {
    await playCopySound();
  };

  return (
    <ControlFrame className={cn({ "gap-2": canPreviewSound })}>
      <Switch checked={value} disabled={disabled} onChange={handleChange} />
      {canPreviewSound ? (
        <>
          <Divider type="vertical" />
          <Tooltip title="试听提示音">
            <button
              aria-label="试听提示音"
              className="inline-flex size-8 shrink-0 cursor-pointer items-center justify-center rounded-1.5 border-0 bg-transparent p-0 text-ant-secondary text-base transition-colors hover:bg-ant-fill-quaternary hover:text-ant-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ant-primary disabled:cursor-not-allowed disabled:text-ant-disabled motion-reduce:transition-none"
              disabled={disabled}
              onClick={handlePreviewSound}
              type="button"
            >
              <i aria-hidden="true" className="i-lucide:play" />
            </button>
          </Tooltip>
        </>
      ) : null}
    </ControlFrame>
  );
};

export default SwitchControl;
