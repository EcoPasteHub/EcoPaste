import { useSnapshot } from "valtio";
import { settingsState, updateSettings } from "@/stores/settings";
import type { Appearance, Language, Theme } from "@/types/settings";
import { SelectControl } from "../components/Field";
import Row from "../components/Row";

const patch = (p: Partial<Appearance>) => updateSettings({ appearance: p });

const AppearancePanel = () => {
  const { value } = useSnapshot(settingsState);
  if (!value) return null;
  const a = value.appearance;

  return (
    <div className="flex flex-col divide-y divide-default-100">
      <Row
        control={
          <SelectControl<Theme>
            onChange={(v) => patch({ theme: v })}
            options={[
              { key: "auto", label: "跟随系统" },
              { key: "light", label: "浅色" },
              { key: "dark", label: "深色" },
            ]}
            value={a.theme}
          />
        }
        label="主题"
      />
      <Row
        control={
          <SelectControl<Language>
            onChange={(v) => patch({ language: v })}
            options={[
              { key: "zh-CN", label: "简体中文" },
              { key: "zh-TW", label: "繁體中文" },
              { key: "en-US", label: "English" },
              { key: "ja-JP", label: "日本語" },
            ]}
            value={a.language}
          />
        }
        description="界面文案语言（i18n 接入后即时生效）"
        label="语言"
      />
    </div>
  );
};

export default AppearancePanel;
