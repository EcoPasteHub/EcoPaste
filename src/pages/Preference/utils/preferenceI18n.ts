import type { TFunction } from "i18next";
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
  return t(`schema.tabs.${tab.id}.title`, { defaultValue: tab.title });
}

/**
 * 翻译偏好二级分组字段；key 由稳定 section id 推导。
 */
export function translatePreferenceSection(
  t: PreferenceTranslator,
  section: PreferenceSection,
  field: "description" | "title",
) {
  return t(`schema.sections.${section.id}.${field}`, {
    defaultValue: section[field],
  });
}

/**
 * 翻译偏好设置项字段；key 由稳定 setting id 推导。
 */
export function translatePreferenceSetting(
  t: PreferenceTranslator,
  setting: PreferenceSetting,
  field: "description" | "title",
) {
  return t(`schema.settings.${setting.id}.${field}`, {
    defaultValue: setting[field],
  });
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
    label: t(`schema.settings.${setting.id}.options.${option.value}`, {
      defaultValue: option.label,
    }),
  };
}

/**
 * 翻译 action/status 控件标签；key 由 setting id 推导。
 */
export function translatePreferenceControlLabel(
  t: PreferenceTranslator,
  setting: PreferenceSetting,
) {
  if (setting.control.type !== "action" && setting.control.type !== "status") {
    return "";
  }

  return t(`schema.settings.${setting.id}.controlLabel`, {
    defaultValue: setting.control.label,
  });
}

/**
 * 翻译文本控件 placeholder；key 由 setting id 推导。
 */
export function translatePreferencePlaceholder(
  t: PreferenceTranslator,
  setting: PreferenceSetting,
) {
  if (
    setting.control.type !== "text" &&
    setting.control.type !== "textarea" &&
    setting.control.type !== "number"
  ) {
    return "";
  }

  return t(`schema.settings.${setting.id}.placeholder`, {
    defaultValue: setting.control.placeholder ?? "",
  });
}
