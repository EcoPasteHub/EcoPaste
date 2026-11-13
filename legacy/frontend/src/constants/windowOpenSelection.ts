import type {
  WindowOpenCategorySelection,
  WindowOpenRangeSelection,
} from "@/types/settings";

export const WINDOW_OPEN_SELECTION_PRESERVE = "preserve";
export const WINDOW_OPEN_SELECTION_ALL = "all";
export const WINDOW_OPEN_GROUP_PREFIX = "group:";

export const WINDOW_OPEN_RANGE_OPTIONS = [
  { value: WINDOW_OPEN_SELECTION_PRESERVE },
  { value: WINDOW_OPEN_SELECTION_ALL },
  { value: "favorite" },
] satisfies Array<{ value: WindowOpenRangeSelection }>;

export const WINDOW_OPEN_CATEGORY_OPTIONS = [
  { value: WINDOW_OPEN_SELECTION_PRESERVE },
  { value: WINDOW_OPEN_SELECTION_ALL },
  { value: "text" },
  { value: "image" },
  { value: "files" },
] satisfies Array<{ value: WindowOpenCategorySelection }>;

/**
 * Encodes a custom clipboard group id as a settings select value.
 */
export function toWindowOpenGroupValue(groupId: string) {
  return `${WINDOW_OPEN_GROUP_PREFIX}${groupId}`;
}

/**
 * Decodes a custom group select value; built-in selections return null.
 */
export function parseWindowOpenGroupId(value: string) {
  if (!value.startsWith(WINDOW_OPEN_GROUP_PREFIX)) return null;

  const groupId = value.slice(WINDOW_OPEN_GROUP_PREFIX.length);
  if (!groupId) return null;

  return groupId;
}
