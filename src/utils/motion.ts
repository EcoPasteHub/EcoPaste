/**
 * 减少动效相关判定的单一入口。
 *
 * `<html>` 上的标记类由 `useReduceMotion` 写入，CSS 抑制规则（`styles/global.scss`）
 * 与命令式 API（如滚动行为）都读它，避免应用开关与系统偏好被组合两次。
 */

/** `<html>` 上的减少动效标记类；改动时需同步 `src/styles/global.scss` 中的同名选择器。 */
export const REDUCE_MOTION_CLASS = "reduce-motion";

/**
 * 读取当前是否处于减少动效状态。
 * 供事件回调这类非渲染路径使用；渲染路径直接用 `useReduceMotion` 的返回值。
 */
export const isReduceMotionEnabled = () => {
  return document.documentElement.classList.contains(REDUCE_MOTION_CLASS);
};
