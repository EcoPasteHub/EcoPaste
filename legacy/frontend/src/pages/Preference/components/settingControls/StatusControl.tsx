import { Switch, Tag } from "antd";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import type { PreferenceSetting } from "../../types/preferences";
import { translatePreferenceControlLabel } from "../../utils/preferenceI18n";
import ControlFrame from "./ControlFrame";

interface StatusControlProps {
  setting: PreferenceSetting;
}

/**
 * 展示不可配置的能力状态；默认能力用只读开关表达，未来能力用标签表达。
 */
const StatusControl: FC<StatusControlProps> = (props) => {
  const { t } = useTranslation("preferences");
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
      <Tag>{translatePreferenceControlLabel(t, setting)}</Tag>
    </ControlFrame>
  );
};

export default StatusControl;
