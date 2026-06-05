import { Switch } from "antd";
import type { FC } from "react";
import type { PreferenceSetting } from "../../types/preferences";
import ControlFrame from "./ControlFrame";

interface StatusControlProps {
  setting: PreferenceSetting;
}

/**
 * 展示不可配置的能力状态；默认能力用只读开关表达，未来能力用标签表达。
 */
const StatusControl: FC<StatusControlProps> = (props) => {
  const { setting } = props;

  if (setting.status === "alwaysOn") {
    return (
      <ControlFrame>
        <Switch checked disabled />
      </ControlFrame>
    );
  }

  return (
    <ControlFrame>
      <span className="inline-flex h-8 w-28 items-center justify-center gap-1 whitespace-nowrap rounded-1.5 border border-ant-border-secondary bg-ant-fill-quaternary px-2.5 text-ant-secondary text-xs leading-none">
        {setting.control.type === "status" ? setting.control.label : ""}
      </span>
    </ControlFrame>
  );
};

export default StatusControl;
