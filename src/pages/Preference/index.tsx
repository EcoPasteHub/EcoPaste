import { Tabs } from "antd";
import { useTranslation } from "react-i18next";
import AboutPanel from "./panels/AboutPanel";
import AppearancePanel from "./panels/AppearancePanel";
import AppFilterPanel from "./panels/AppFilterPanel";
import ClipboardPanel from "./panels/ClipboardPanel";
import GeneralPanel from "./panels/GeneralPanel";
import ShortcutsPanel from "./panels/ShortcutsPanel";

type GroupKey =
  | "general"
  | "clipboard"
  | "filters"
  | "shortcuts"
  | "appearance"
  | "about";

const GROUPS: { key: GroupKey; panel: React.ReactNode }[] = [
  { key: "general", panel: <GeneralPanel /> },
  { key: "clipboard", panel: <ClipboardPanel /> },
  { key: "filters", panel: <AppFilterPanel /> },
  { key: "shortcuts", panel: <ShortcutsPanel /> },
  { key: "appearance", panel: <AppearancePanel /> },
  { key: "about", panel: <AboutPanel /> },
];

/**
 * 左侧导航 + 右侧面板：antd `Tabs tabPosition="left"`，items 内 children 即面板内容。
 */
const Preference = () => {
  const { t } = useTranslation();
  return (
    <Tabs
      className="h-screen w-screen [&_.ant-tabs-nav]:w-36 [&_.ant-tabs-nav]:p-2"
      defaultActiveKey="general"
      items={GROUPS.map(({ key, panel }) => ({
        children: <div className="flex-1 overflow-auto p-4">{panel}</div>,
        key,
        label: t(`preference.tab.${key}`),
      }))}
      tabPosition="left"
    />
  );
};

export default Preference;
