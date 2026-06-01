import { convertFileSrc } from "@tauri-apps/api/core";
import type { FC } from "react";
import { useMemo } from "react";
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
    return (
      <img
        alt={singleImageFile.path}
        className="max-h-21"
        src={convertFileSrc(singleImageFile.path)}
      />
    );
  }

  return (
    <div className="flex flex-col gap-1">
      {visibleEntries.map((entry) => (
        <div
          className="flex items-center gap-1 truncate"
          key={`${entry.path}-${entry.type}`}
          title={entry.path}
        >
          <img
            alt=""
            className="size-5 shrink-0 rounded-0.5"
            src={convertFileSrc(entry.iconPath)}
          />

          <span className="truncate">{entry.path}</span>
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

export default FilesCard;
