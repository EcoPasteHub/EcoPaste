import { Tabs } from "@heroui/react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import AboutPanel from "./panels/AboutPanel";
import AppearancePanel from "./panels/AppearancePanel";
import ClipboardPanel from "./panels/ClipboardPanel";
import FiltersPanel from "./panels/FiltersPanel";
import GeneralPanel from "./panels/GeneralPanel";
import ShortcutsPanel from "./panels/ShortcutsPanel";

type GroupKey =
  | "general"
  | "clipboard"
  | "filters"
  | "shortcuts"
  | "appearance"
  | "about";

const GROUPS: { key: GroupKey; panel: ReactNode }[] = [
  { key: "general", panel: <GeneralPanel /> },
  { key: "clipboard", panel: <ClipboardPanel /> },
  { key: "filters", panel: <FiltersPanel /> },
  { key: "shortcuts", panel: <ShortcutsPanel /> },
  { key: "appearance", panel: <AppearancePanel /> },
  { key: "about", panel: <AboutPanel /> },
];

const Preference = () => {
  const { t } = useTranslation();
  return (
    <Tabs
      className="flex h-screen w-screen gap-0"
      defaultSelectedKey="general"
      orientation="vertical"
    >
      <Tabs.ListContainer className="w-36 border-default-200 border-r">
        <Tabs.List aria-label={t("preference.title")} className="p-2">
          {GROUPS.map(({ key }) => (
            <Tabs.Tab id={key} key={key}>
              {t(`preference.tab.${key}`)}
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
