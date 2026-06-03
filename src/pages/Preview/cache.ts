import type { ClipboardPreviewPayload } from "@/commands";
import { PREVIEW_CACHE_LIMIT } from "./constants";

/**
 * 从 LRU cache 读取同 item 的最新 payload，命中后刷新插入顺序。
 */
export function readCachedPayload(
  cache: Map<string, ClipboardPreviewPayload>,
  itemId: string,
) {
  const key = [...cache.keys()].find((entryKey) => {
    return entryKey.startsWith(`${itemId}:`);
  });

  if (!key) return null;

  const cached = cache.get(key) ?? null;
  if (cached) {
    cache.delete(key);
    cache.set(key, cached);
  }

  return cached;
}

/**
 * 写入最近预览 payload，key 绑定 updatedAt 避免内容复用过期。
 */
export function writeCachedPayload(
  cache: Map<string, ClipboardPreviewPayload>,
  nextPayload: ClipboardPreviewPayload,
) {
  cache.set(cacheKey(nextPayload), nextPayload);

  while (cache.size > PREVIEW_CACHE_LIMIT) {
    const [oldestKey] = cache.keys();
    if (!oldestKey) return;

    cache.delete(oldestKey);
  }
}

/**
 * 生成预览 payload 的缓存 key。
 */
export function cacheKey(payload: ClipboardPreviewPayload) {
  return `${payload.id}:${payload.updatedAt}`;
}
