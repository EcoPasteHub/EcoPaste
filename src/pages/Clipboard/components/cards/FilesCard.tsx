import type { FC } from "react";
import { useMemo } from "react";
import AssetImage from "@/components/AssetImage";
import type { ClipboardItem } from "@/types/clipboard";
import { isImage } from "@/utils/is";

/**
 * 文件类卡片：`content` 是路径列表（用 \n 分隔），`file_types` 是同序的 d/f 标记。
 * 图标路径由 Rust 命令层预处理返回到 `fileIconPaths`，这里直接渲染，最多显示前三项。
 */
const FilesCard: FC<ClipboardItem> = (props) => {
  const entries = useMemo(() => parseFiles(props), [props]);
  const visibleEntries = entries.slice(0, 3);
  const singleImageFile = useMemo(() => getSingleImageFile(entries), [entries]);

  if (singleImageFile) {
    return <AssetImage className="max-h-21" src={singleImageFile.path} />;
  }

  return (
    <div className="flex flex-col gap-1">
      {visibleEntries.map((entry) => (
        <div
          className="flex items-center gap-1 truncate"
          key={`${entry.path}-${entry.type}`}
          title={entry.path}
        >
          <AssetImage className="size-5" src={entry.iconPath} />

          <span className="truncate">{getFileName(entry.path)}</span>
        </div>
      ))}
    </div>
  );
};

interface FileEntry {
  path: string;
  type: "d" | "f";
  iconPath: string | null;
}

const parseFiles = (item: ClipboardItem): FileEntry[] => {
  const paths = item.content.split("\n").filter(Boolean);
  const types = (item.fileTypes ?? "").split(",");
  const iconPaths = parseFileIconPaths(item.fileIconPaths);

  return paths.map((path, idx) => ({
    iconPath: iconPaths[idx] ?? null,
    path,
    type: types[idx] === "d" ? "d" : "f",
  }));
};

const parseFileIconPaths = (raw?: string): Array<string | null> => {
  if (!raw) return [];

  try {
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];

    return parsed.map((entry) => (typeof entry === "string" ? entry : null));
  } catch {
    return [];
  }
};

const getSingleImageFile = (entries: FileEntry[]): FileEntry | null => {
  if (entries.length !== 1) return null;

  const [entry] = entries;

  if (entry.type !== "f") return null;

  if (!isImage(entry.path)) return null;

  return entry;
};

/**
 * 从完整路径中取出最后一级名称，兼容 macOS 的 `/` 和 Windows 的 `\`。
 */
const getFileName = (path: string) => {
  const separatorIndex = Math.max(
    path.lastIndexOf("/"),
    path.lastIndexOf("\\"),
  );

  if (separatorIndex < 0) return path;

  return path.slice(separatorIndex + 1);
};

export default FilesCard;
