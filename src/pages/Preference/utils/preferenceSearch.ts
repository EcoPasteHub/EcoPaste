import { allPreferenceSettings } from "../config/preferenceSchema";

export type PreferenceSearchResult = (typeof allPreferenceSettings)[number];

/**
 * 从完整设置 schema 中执行轻量本地搜索，返回最多 8 条可跳转结果。
 */
export function searchPreferenceSettings(
  query: string,
): PreferenceSearchResult[] {
  const normalized = query.trim().toLowerCase();
  if (!normalized) return [];

  return allPreferenceSettings
    .filter(({ section, setting, tab }) => {
      const haystack = [
        tab.title,
        section.title,
        setting.title,
        setting.description,
        ...(setting.keywords ?? []),
      ]
        .join(" ")
        .toLowerCase();

      return haystack.includes(normalized);
    })
    .slice(0, 8);
}
