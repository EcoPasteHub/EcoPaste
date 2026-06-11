import { listen } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { proxy } from "valtio";

import { TAURI_EVENT } from "@/constants/events";
import { log } from "@/utils/log";

/**
 * 窗口生命周期阶段，与 Rust `LifecyclePhase` 一一对应（camelCase）。
 */
export type LifecyclePhase =
  | "created"
  | "ready"
  | "visible"
  | "hiddenWarm"
  | "dormant"
  | "destroyPending"
  | "destroyed";

/**
 * Rust `window://lifecycle` 事件 payload。
 */
interface LifecyclePayload {
  label: string;
  phase: LifecyclePhase;
  generation: number;
  reason: string;
  visible: boolean;
}

/**
 * 当前 WebView 的生命周期镜像。真相源在 Rust（`WindowLifecycleManager`）。
 *
 * 每个 WebView 只关心自身窗口的阶段，故事件订阅按 `label` 过滤；跨窗口的阶段不进本镜像。
 * 初值 `created` 仅占位，Rust 在 show / hide / ready 时通过事件推进。
 */
export const windowLifecycleState = proxy<{
  phase: LifecyclePhase;
  visible: boolean;
  generation: number;
}>({
  generation: 0,
  phase: "created",
  visible: false,
});

const currentLabel = getCurrentWebviewWindow().label;

/**
 * 启动期一次性订阅 `window://lifecycle`，只接收当前窗口的阶段更新。
 * 模块导入即开跑，单 WebView 内天然单例。
 */
export const windowLifecycleReady: Promise<void> = (async () => {
  try {
    await listen<LifecyclePayload>(TAURI_EVENT.WINDOW_LIFECYCLE, (event) => {
      const { generation, label, phase, visible } = event.payload;

      if (label !== currentLabel) return;

      windowLifecycleState.generation = generation;
      windowLifecycleState.phase = phase;
      windowLifecycleState.visible = visible;
    });
  } catch (error) {
    log.error("Failed to listen window lifecycle event", error);
  }
})();
