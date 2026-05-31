import { cn } from "@heroui/styles";
import { convertFileSrc, invoke } from "@tauri-apps/api/core";
import { useEffect, useState } from "react";

import type { ClipboardItem } from "@/types/clipboard";

const MAX_VISIBLE = 3;

// 进程级 icon 路径缓存：避免列表来回滚动时重复 IPC。
// 只缓存 icon 的 src（按类型决定，稳定），不缓存 exists（路径状态会动态变化）。
const iconCache = new Map<string, string | null>();
const cacheKey = (path: string, index: number, fileTypes?: string | null) =>
  `${fileTypes ?? ""}|${index}|${path}`;

interface FileIconResult {
  iconPath: string | null;
  exists: boolean;
}

// `content` 为换行分隔的绝对路径列表（见 src-tauri/src/clipboard/ingest.rs:49-50）。
const parsePaths = (content: string): string[] =>
  content.split("\n").filter(Boolean);

const basename = (p: string): string => {
  const idx = Math.max(p.lastIndexOf("/"), p.lastIndexOf("\\"));
  return idx >= 0 ? p.slice(idx + 1) : p;
};

const FileRow = ({
  path,
  index,
  fileTypes,
}: {
  path: string;
  index: number;
  fileTypes?: string | null;
}) => {
  const key = cacheKey(path, index, fileTypes);
  const [iconSrc, setIconSrc] = useState<string | null>(
    () => iconCache.get(key) ?? null,
  );
  const [exists, setExists] = useState<boolean>(true);

  useEffect(() => {
    invoke<FileIconResult>("get_file_icon_path", {
      fileTypes,
      index,
      path,
    })
      .then((result) => {
        const src = result.iconPath ? convertFileSrc(result.iconPath) : null;
        iconCache.set(key, src);
        setIconSrc(src);
        setExists(result.exists);
      })
      .catch(() => {
        iconCache.set(key, null);
      });
  }, [key, path, index, fileTypes]);

  return (
    <div
      className={cn("flex min-w-0 items-center gap-2", {
        "opacity-50": !exists,
      })}
      title={exists ? path : `${path}（文件已不存在）`}
    >
      {iconSrc && <img alt="" className="size-4 shrink-0" src={iconSrc} />}
      <span className={cn("truncate text-sm", { "line-through": !exists })}>
        {basename(path)}
      </span>
    </div>
  );
};

const FilesCard = ({ item }: { item: ClipboardItem }) => {
  const paths = parsePaths(item.content);
  const visible = paths.slice(0, MAX_VISIBLE);
  const overflow = paths.length - visible.length;

  return (
    <div className="min-w-0">
      <div className="text-muted text-xs">
        Files{paths.length > 1 ? ` · ${paths.length}` : ""}
      </div>
      <div className="flex flex-col gap-1">
        {visible.map((path, index) => (
          <FileRow
            fileTypes={item.fileTypes}
            index={index}
            key={path}
            path={path}
          />
        ))}
        {overflow > 0 ? (
          <div className="text-muted text-xs">等 {overflow} 项</div>
        ) : null}
      </div>
    </div>
  );
};

export default FilesCard;
