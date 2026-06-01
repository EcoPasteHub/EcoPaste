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

  if (entries.length === 0) {
    return <span className="text-gray-400">(无文件)</span>;
  }

  if (singleImageFile) {
    return (
      <div className="flex items-center gap-2">
        <img
          alt={singleImageFile.path}
          className="size-16 shrink-0 rounded-1.5 object-cover"
          src={convertFileSrc(singleImageFile.path)}
        />
        <span className="truncate text-sm" title={singleImageFile.path}>
          {singleImageFile.path}
        </span>
      </div>
    );
  }

  return (
    <ul className="m-0 flex list-none flex-col gap-0.5 p-0">
      {visibleEntries.map((entry) => (
        <li
          className="flex items-center gap-1 truncate text-sm"
          key={`${entry.path}-${entry.type}`}
          title={entry.path}
        >
          {entry.iconPath ? (
            <img
              alt=""
              className="size-4 shrink-0 rounded-0.5"
              src={convertFileSrc(entry.iconPath)}
            />
          ) : (
            <span className="text-gray-400">
              {entry.type === "d" ? "📁" : "📄"}
            </span>
          )}
          <span className="truncate">{entry.path}</span>
        </li>
      ))}
    </ul>
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
