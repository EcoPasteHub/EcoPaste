import { setAutostart, showTaskbarIcon } from "@/commands";
import { updateSettings } from "@/stores/settings";
import type { SettingsPatch } from "@/types/settings";
import type { PreferenceSetting, SettingValue } from "../types/preferences";

/**
 * 提交单个设置变更，同时处理与该设置绑定的系统副作用。
 */
export async function commitSettingChange(
  setting: PreferenceSetting,
  value: SettingValue,
) {
  if (!setting.path) return;

  await applySideEffects(setting, value);
  await updateSettings(buildPatch(setting.path, value));
}

/**
 * 比较设置值是否真正发生变化，避免无意义保存。
 */
export function settingValuesEqual(left: SettingValue, right: SettingValue) {
  if (Array.isArray(left) || Array.isArray(right)) {
    if (!Array.isArray(left) || !Array.isArray(right)) return false;
    if (left.length !== right.length) return false;

    return left.every((item, index) => {
      return item === right[index];
    });
  }

  if (typeof left === "object" || typeof right === "object") {
    if (typeof left !== "object" || typeof right !== "object") return false;
    if (left === null || right === null) return false;

    return JSON.stringify(left) === JSON.stringify(right);
  }

  return left === right;
}

/**
 * 把 schema path 和控件值转换成 Rust `update_settings` 可接收的深层补丁。
 */
function buildPatch(
  path: readonly string[],
  value: SettingValue,
): SettingsPatch {
  const [head, ...rest] = path;
  if (!head) return {};

  return {
    [head]: buildNestedPatch(rest, value),
  } as SettingsPatch;
}

/**
 * 递归构造嵌套补丁，数组字段保持整体替换语义。
 */
function buildNestedPatch(
  path: readonly string[],
  value: SettingValue,
): SettingValue | Record<string, unknown> {
  const [head, ...rest] = path;
  if (!head) return value;

  return {
    [head]: buildNestedPatch(rest, value),
  };
}

/**
 * 对需要立即触达系统层的设置执行配套副作用。
 */
async function applySideEffects(
  setting: PreferenceSetting,
  value: SettingValue,
) {
  if (setting.id === "control.autoStart") {
    await setAutostart(Boolean(value));
    return;
  }

  if (setting.id === "control.dockIcon") {
    await showTaskbarIcon(Boolean(value));
  }
}
