import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import { settingsState, updateSettings } from "@/stores/settings";
import type { General } from "@/types/settings";
import { Toggle } from "../components/Field";
import Row from "../components/Row";

const patch = (p: Partial<General>) => updateSettings({ general: p });

const FIELDS: (keyof General)[] = [
  "autoStart",
  "silentStart",
  "trayIcon",
  "dockIcon",
];

const GeneralPanel = () => {
  const { t } = useTranslation();
  const { value } = useSnapshot(settingsState);
  if (!value) return null;
  const g = value.general;

  return (
    <div className="flex flex-col divide-y divide-border-secondary">
      {FIELDS.map((field) => (
        <Row
          control={
            <Toggle
              isSelected={g[field]}
              onChange={(v) => patch({ [field]: v })}
            />
          }
          description={t(`general.${field}.desc`)}
          key={field}
          label={t(`general.${field}.label`)}
        />
      ))}
    </div>
  );
};

export default GeneralPanel;
