import { useSnapshot } from "valtio";
import { settingsState, updateSettings } from "@/stores/settings";
import type { General } from "@/types/settings";
import { Toggle } from "../components/Field";
import Row from "../components/Row";

const patch = (p: Partial<General>) => updateSettings({ general: p });

const GeneralPanel = () => {
  const { value } = useSnapshot(settingsState);
  if (!value) return null;
  const g = value.general;

  return (
    <div className="flex flex-col divide-y divide-default-100">
      <Row
        control={
          <Toggle
            isSelected={g.autoStart}
            onChange={(v) => patch({ autoStart: v })}
          />
        }
        description="登录系统后自动启动 EcoPaste"
        label="开机自启"
      />
      <Row
        control={
          <Toggle
            isSelected={g.silentStart}
            onChange={(v) => patch({ silentStart: v })}
          />
        }
        description="开机自启后不弹出主窗，仅驻留托盘"
        label="静默启动"
      />
      <Row
        control={
          <Toggle
            isSelected={g.trayIcon}
            onChange={(v) => patch({ trayIcon: v })}
          />
        }
        description="macOS 菜单栏 / Windows 系统托盘"
        label="显示托盘图标"
      />
      <Row
        control={
          <Toggle
            isSelected={g.dockIcon}
            onChange={(v) => patch({ dockIcon: v })}
          />
        }
        description="macOS Dock / Windows 任务栏"
        label="显示 Dock 图标"
      />
    </div>
  );
};

export default GeneralPanel;
