import { useReducedMotion } from "motion/react";
import { useSnapshot } from "valtio";
import { settingsState } from "@/stores/settings";

/**
 * 综合「应用内减少动画效果开关」与「系统减少动态效果偏好」的当前状态。
 *
 * motion 组件用它裁掉位移 / 缩放 / 布局动画；`<html>` 上的 `reduce-motion` 类由
 * `App` 依据同一个值写入，供 CSS 与 `isReduceMotionEnabled()` 共用同一判定。
 */
export const useReduceMotion = (): boolean => {
  const settings = useSnapshot(settingsState);
  const systemPrefersReduce = useReducedMotion();

  return settings.appearance.reduceMotion || systemPrefersReduce === true;
};
