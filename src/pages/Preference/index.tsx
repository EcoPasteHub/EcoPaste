import { Tabs } from "@heroui/react";
import type { ReactNode } from "react";
import AboutPanel from "./panels/AboutPanel";
import AppearancePanel from "./panels/AppearancePanel";
import ClipboardPanel from "./panels/ClipboardPanel";
import GeneralPanel from "./panels/GeneralPanel";
import ShortcutsPanel from "./panels/ShortcutsPanel";

type GroupKey = "general" | "clipboard" | "shortcuts" | "appearance" | "about";

const GROUPS: { key: GroupKey; label: string; panel: ReactNode }[] = [
  { key: "general", label: "常规", panel: <GeneralPanel /> },
  { key: "clipboard", label: "剪贴板", panel: <ClipboardPanel /> },
  { key: "shortcuts", label: "快捷键", panel: <ShortcutsPanel /> },
  { key: "appearance", label: "外观", panel: <AppearancePanel /> },
  { key: "about", label: "关于", panel: <AboutPanel /> },
];

const Preference = () => {
  return (
    <Tabs
      className="flex h-screen w-screen gap-0"
      defaultSelectedKey="general"
      orientation="vertical"
    >
      <Tabs.ListContainer className="w-36 border-default-200 border-r">
        <Tabs.List aria-label="偏好设置" className="p-2">
          {GROUPS.map(({ key, label }) => (
            <Tabs.Tab id={key} key={key}>
              {label}
              <Tabs.Indicator />
            </Tabs.Tab>
          ))}
        </Tabs.List>
      </Tabs.ListContainer>
      {GROUPS.map(({ key, panel }) => (
        <Tabs.Panel className="flex-1 overflow-auto p-4" id={key} key={key}>
          {panel}
        </Tabs.Panel>
      ))}
    </Tabs>
  );
};

export default Preference;
