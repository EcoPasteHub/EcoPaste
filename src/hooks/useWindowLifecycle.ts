import { useSnapshot } from "valtio";

import { windowLifecycleState } from "@/stores/windowLifecycle";

/**
 * 读取当前 WebView 的生命周期镜像（阶段 / 可见性 / 代次）。
 * 后续 Phase 2 / 3 的冻结、销毁 gate 都从这里取当前窗口状态。
 */
export const useWindowLifecycle = () => {
  return useSnapshot(windowLifecycleState);
};
