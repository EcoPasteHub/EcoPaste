import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import { settingsState, updateSettings } from "@/stores/settings";
import type { Shortcuts } from "@/types/settings";
import Row from "../components/Row";
import ShortcutInput from "../components/ShortcutInput";

const patch = (p: Partial<Shortcuts>) => updateSettings({ shortcuts: p });

const ShortcutsPanel = () => {
  const { t } = useTranslation();
  const { value } = useSnapshot(settingsState);
  if (!value) return null;
  const s = value.shortcuts;

  return (
    <div className="flex flex-col divide-y divide-default-100">
      <Row
        control={
          <ShortcutInput
            onChange={(v) => patch({ openClipboard: v })}
            value={s.openClipboard}
          />
        }
        description={t("shortcuts.openClipboard.desc")}
        label={t("shortcuts.openClipboard.label")}
      />
      <Row
        control={
          <ShortcutInput
            onChange={(v) => patch({ openPreference: v })}
            value={s.openPreference}
          />
        }
        label={t("shortcuts.openPreference.label")}
      />
      <Row
        control={
          <ShortcutInput
            onChange={(v) => patch({ pastePlain: v })}
            value={s.pastePlain}
          />
        }
        description={t("shortcuts.pastePlain.desc")}
        label={t("shortcuts.pastePlain.label")}
      />
      <Row
        control={
          <ShortcutInput
            modifierOnly
            onChange={(v) =>
              patch({ quickPaste: { ...s.quickPaste, modifier: v } })
            }
            value={s.quickPaste.modifier}
          />
        }
        description={t("shortcuts.quickPasteModifier.desc")}
        label={t("shortcuts.quickPasteModifier.label")}
      />
    </div>
  );
};

export default ShortcutsPanel;
