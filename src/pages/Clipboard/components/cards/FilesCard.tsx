import { cn } from "@heroui/styles";
import { convertFileSrc, invoke } from "@tauri-apps/api/core";
import { useEffect, useState } from "react";

import type { ClipboardItem } from "@/types/clipboard";
import { isImage } from "@/utils/is";

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

// 单个图片文件用 ImageCard 同款样式展示。文件被删时缩略图加载失败 →
// 由父组件捕获 onError 切回 FileRow 渲染，避免显示破图。
const ImagePreview = ({
  path,
  onError,
}: {
  path: string;
  onError: () => void;
}) => {
  const [dims, setDims] = useState<string>("");
  const src = convertFileSrc(path);

  return (
    <div className="flex min-w-0 items-start gap-2">
      <img
        alt=""
        className="size-12 shrink-0 rounded border border-separator object-cover"
        onError={onError}
        onLoad={(e) => {
          const img = e.currentTarget;
          setDims(`${img.naturalWidth}×${img.naturalHeight}`);
        }}
        src={src}
      />
      <div className="min-w-0 flex-1">
        <div className="text-muted text-xs">Image · File</div>
        <div className="truncate text-foreground text-sm">{basename(path)}</div>
        {dims ? <div className="text-muted text-xs">{dims}</div> : null}
      </div>
    </div>
  );
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
  const [imageLoadFailed, setImageLoadFailed] = useState(false);

  // 单个图片文件（非目录、扩展名匹配）走 ImageCard 同款样式；
  // 加载失败（文件被删）则退回 FileRow 渲染，避免显示破图。
  const isSingleImage =
    paths.length === 1 &&
    item.fileTypes?.split(",")[0] !== "d" &&
    isImage(paths[0]) &&
    !imageLoadFailed;

  if (isSingleImage) {
    return (
      <ImagePreview onError={() => setImageLoadFailed(true)} path={paths[0]} />
    );
  }

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
