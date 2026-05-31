import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import { settingsState, updateSettings } from "@/stores/settings";
import type { Appearance, Language, Theme } from "@/types/settings";
import { SelectControl } from "../components/Field";
import Row from "../components/Row";

const patch = (p: Partial<Appearance>) => updateSettings({ appearance: p });

const THEME_KEYS: Theme[] = ["auto", "light", "dark"];

const LANG_KEYS: Language[] = ["zh-CN", "en-US"];
const LANG_LABEL: Record<Language, string> = {
  "en-US": "English",
  "zh-CN": "简体中文",
};

const AppearancePanel = () => {
  const { t } = useTranslation();
  const { value } = useSnapshot(settingsState);
  if (!value) return null;
  const a = value.appearance;

  return (
    <div className="flex flex-col divide-y divide-border-secondary">
      <Row
        control={
          <SelectControl<Theme>
            onChange={(v) => patch({ theme: v })}
            options={THEME_KEYS.map((key) => ({
              key,
              label: t(`appearance.theme.option.${key}`),
            }))}
            value={a.theme}
          />
        }
        label={t("appearance.theme.label")}
      />
      <Row
        control={
          <SelectControl<Language>
            onChange={(v) => patch({ language: v })}
            options={LANG_KEYS.map((key) => ({ key, label: LANG_LABEL[key] }))}
            value={a.language}
          />
        }
        description={t("appearance.language.desc")}
        label={t("appearance.language.label")}
      />
    </div>
  );
};

export default AppearancePanel;
