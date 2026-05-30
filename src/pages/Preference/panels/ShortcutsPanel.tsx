import { useSnapshot } from "valtio";
import { settingsState, updateSettings } from "@/stores/settings";
import type { Shortcuts } from "@/types/settings";
import Row from "../components/Row";
import ShortcutInput from "../components/ShortcutInput";

const patch = (p: Partial<Shortcuts>) => updateSettings({ shortcuts: p });

const ShortcutsPanel = () => {
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
        description="显示/隐藏主窗"
        label="唤起剪贴板"
      />
      <Row
        control={
          <ShortcutInput
            onChange={(v) => patch({ openPreference: v })}
            value={s.openPreference}
          />
        }
        label="打开偏好设置"
      />
      <Row
        control={
          <ShortcutInput
            onChange={(v) => patch({ pastePlain: v })}
            value={s.pastePlain}
          />
        }
        description="主窗内局部生效（不在 OS 级注册）"
        label="粘贴时去除格式"
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
        description="按住此修饰键 + 数字键粘贴第 N 条"
        label="快速粘贴修饰键"
      />
    </div>
  );
};

export default ShortcutsPanel;
