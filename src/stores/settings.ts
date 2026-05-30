// 前端只持有设置的本地镜像，真相源仍在 Rust。
// 任何变更必须经 `update_settings` 命令写回，再用返回的快照覆盖镜像，避免本地状态漂移。

import { invoke } from "@tauri-apps/api/core";
import { proxy } from "valtio";
import { TAURI_COMMAND } from "@/constants/commands";
import type { Settings, SettingsPatch } from "@/types/settings";

interface SettingsState {
  // 启动期未加载完成时为 null；组件层用 `loaded` 判定是否可读字段。
  value: Settings | null;
  loaded: boolean;
}

export const settingsState = proxy<SettingsState>({
  loaded: false,
  value: null,
});

export async function loadSettings(): Promise<Settings> {
  const next = await invoke<Settings>(TAURI_COMMAND.GET_SETTINGS);
  settingsState.value = next;
  settingsState.loaded = true;
  return next;
}

export async function updateSettings(patch: SettingsPatch): Promise<Settings> {
  const next = await invoke<Settings>(TAURI_COMMAND.UPDATE_SETTINGS, { patch });
  settingsState.value = next;
  settingsState.loaded = true;
  return next;
}
