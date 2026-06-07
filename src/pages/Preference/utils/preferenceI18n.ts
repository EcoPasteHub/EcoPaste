import type { TFunction } from "i18next";
import { isMac, isWin } from "@/utils/is";
import type {
  PreferenceOption,
  PreferenceSection,
  PreferenceSetting,
  PreferenceTab,
} from "../types/preferences";

type PreferenceTranslator = TFunction<"preferences">;

/**
 * 翻译偏好一级分类标题；key 由稳定 tab id 推导。
 */
export function translatePreferenceTab(
  t: PreferenceTranslator,
  tab: PreferenceTab,
) {
  return t(`schema.tabs.${tab.id}.title`);
}

/**
 * 翻译偏好二级分组字段；key 由稳定 section id 推导。
 */
export function translatePreferenceSection(
  t: PreferenceTranslator,
  section: PreferenceSection,
  field: "description" | "title",
) {
  return t(`schema.sections.${section.id}.${field}`);
}

/**
 * 翻译偏好设置项字段；key 由稳定 setting id 推导。
 */
export function translatePreferenceSetting(
  t: PreferenceTranslator,
  setting: PreferenceSetting,
  field: "description" | "title",
) {
  const platformField = resolvePlatformPreferenceField(setting, field);
  if (platformField) {
    return t(platformField);
  }

  return t(`schema.settings.${setting.id}.${field}`);
}

/**
 * 返回平台特化设置文案 key；仅处理确实有平台差异的设置项。
 */
function resolvePlatformPreferenceField(
  setting: PreferenceSetting,
  field: "description" | "title",
) {
  const platform = isMac ? "macos" : isWin ? "windows" : null;
  if (!platform) return null;

  if (
    setting.id !== "control.autoStart" &&
    setting.id !== "control.trayIcon" &&
    setting.id !== "control.dockIcon"
  ) {
    return null;
  }

  return `schema.settings.${setting.id}.${platform}.${field}`;
}

/**
 * 翻译设置项控件里展示的选项；key 由 setting id + option value 推导。
 */
export function translatePreferenceOption(
  t: PreferenceTranslator,
  setting: PreferenceSetting,
  option: PreferenceOption,
) {
  return {
    ...option,
    label: t(`schema.settings.${setting.id}.options.${option.value}`),
  };
}

/**
 * 翻译带按钮/状态文案的控件标签；key 由 setting id 推导。
 */
export function translatePreferenceControlLabel(
  t: PreferenceTranslator,
  setting: PreferenceSetting,
) {
  if (
    setting.control.type !== "action" &&
    setting.control.type !== "status" &&
    setting.control.type !== "sortableTree" &&
    setting.control.type !== "sortableCheckboxTree"
  ) {
    return "";
  }

  return t(`schema.settings.${setting.id}.controlLabel`);
}

/**
 * 翻译文本控件 placeholder；key 由 setting id 推导。
 */
export function translatePreferencePlaceholder(
  t: PreferenceTranslator,
  setting: PreferenceSetting,
) {
  if (setting.control.type !== "text" && setting.control.type !== "textarea") {
    return "";
  }

  return t(`schema.settings.${setting.id}.placeholder`);
}

/**
 * 翻译数字控件后缀；有显式 suffixKey 时走公共单位表，否则回退到设置项本地默认值。
 */
export function translatePreferenceNumberSuffix(
  t: PreferenceTranslator,
  setting: PreferenceSetting,
) {
  if (setting.control.type !== "number") return "";

  if (setting.control.suffixKey) {
    return t(`schema.numberSuffixes.${setting.control.suffixKey}`);
  }

  return "";
}

/**
 * 翻译只读快捷键说明；key 由 setting id + 快捷键序号推导。
 */
export function translatePreferenceShortcutLabel(
  t: PreferenceTranslator,
  setting: PreferenceSetting,
  shortcutIndex: number,
) {
  if (setting.control.type !== "shortcutTags") return "";

  return t(`schema.settings.${setting.id}.shortcuts.${shortcutIndex}.label`);
}
