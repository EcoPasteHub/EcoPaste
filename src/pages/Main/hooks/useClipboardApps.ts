import { convertFileSrc, invoke } from "@tauri-apps/api/core";
import { useEffect, useRef, useState } from "react";

import type { ClipboardApp, ClipboardItem } from "@/types/clipboard";
import { log } from "@/utils/log";

export interface AppMeta {
  name: string;
  // 图标绝对路径经 convertFileSrc 转换后的 asset URL；无图标为 null。
  iconSrc: string | null;
}

// 卡片列表里需要展示来源应用图标——按 sourceAppId 去重批量拉一次 meta，
// 写入 Map 缓存；后续 items 增量只拉缺失项，避免每条卡片单独 invoke。
export const useClipboardApps = (
  items: ClipboardItem[],
): Map<string, AppMeta> => {
  const [apps, setApps] = useState<Map<string, AppMeta>>(new Map());
  const cacheRef = useRef(apps);
  cacheRef.current = apps;

  useEffect(() => {
    const wanted = new Set<string>();
    for (const it of items) {
      if (it.sourceAppId) wanted.add(it.sourceAppId);
    }
    const missing = [...wanted].filter((id) => !cacheRef.current.has(id));
    if (missing.length === 0) return;

    let cancelled = false;
    (async () => {
      try {
        const list = await invoke<ClipboardApp[]>("list_clipboard_apps", {
          ids: missing,
        });
        const entries = await Promise.all(
          list.map(async (app): Promise<[string, AppMeta]> => {
            let iconSrc: string | null = null;
            if (app.iconFile) {
              try {
                const path = await invoke<string>(
                  "get_clipboard_app_icon_path",
                  { fileName: app.iconFile },
                );
                iconSrc = convertFileSrc(path);
              } catch (err) {
                log.error("get_clipboard_app_icon_path failed", err);
              }
            }
            return [app.id, { iconSrc, name: app.name }];
          }),
        );
        if (cancelled) return;
        setApps((prev) => {
          const next = new Map(prev);
          for (const [id, meta] of entries) next.set(id, meta);
          return next;
        });
      } catch (err) {
        log.error("list_clipboard_apps failed", err);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [items]);

  return apps;
};
