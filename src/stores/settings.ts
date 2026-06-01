import { listen } from "@tauri-apps/api/event";
import { proxy } from "valtio";

import {
  getSettings,
  updateSettings as invokeUpdateSettings,
} from "@/commands";
import { TAURI_EVENT } from "@/constants/events";
import type { Settings, SettingsPatch } from "@/types/settings";

/**
 * 设置的本地镜像，真相源在 Rust（`SettingsStore`）。
 *
 * 数据流向是**严格单向**的：
 *   组件 → `updateSettings(patch)` → Rust 落盘 → Rust `emit("settings://updated")`
 *   → 全部窗口的 `listen` → `Object.assign(settingsState, payload)` → 组件重渲染
 *
 * 因此 `updateSettings` 本身不修改 `settingsState`——发起方也通过事件回灌，保证
 * 多窗口取到的镜像永远等于 Rust 的最新快照，避免本地乐观更新引发漂移。
 *
 * 字面量初值仅为占位；组件层通过 `use(settingsReady)` 挂起到首屏快照灌入后才会读取。
 */
export const settingsState = proxy<Settings>({} as Settings);

/**
 * 启动期一次性初始化：订阅 Rust 广播 + 拉取首屏快照。
 * 模块导入即开跑，由 React `use(settingsReady)` 在 Suspense 中等待完成。
 * 每个 webview 加载本模块一次，因此事件订阅天然单例。
 */
export const settingsReady: Promise<void> = (async () => {
  await listen<Settings>(TAURI_EVENT.SETTINGS_UPDATED, (event) => {
    Object.assign(settingsState, event.payload);
  });

  const initial = await getSettings();

  Object.assign(settingsState, initial);
})();

/**
 * 提交设置补丁；不在此处更新镜像，等 Rust 广播 `settings://updated` 后由 listen 统一回灌。
 * 返回的快照仅为调用方需要立即拿到结果时使用（如表单关闭前校验）。
 */
export async function updateSettings(patch: SettingsPatch): Promise<Settings> {
  return invokeUpdateSettings(patch);
}
