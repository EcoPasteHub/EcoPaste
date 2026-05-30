import type { ClipboardItem } from "@/types/clipboard";

// `content` 为换行分隔的绝对路径列表（见 src-tauri/src/clipboard/ingest.rs:49-50）。
const parsePaths = (content: string): string[] =>
  content.split("\n").filter(Boolean);

const basename = (p: string): string => {
  const idx = Math.max(p.lastIndexOf("/"), p.lastIndexOf("\\"));
  return idx >= 0 ? p.slice(idx + 1) : p;
};

const FilesCard = ({ item }: { item: ClipboardItem }) => {
  const paths = parsePaths(item.content);
  const first = paths[0] ?? "";
  const rest = paths.length - 1;

  return (
    <div className="min-w-0">
      <div className="text-muted text-xs">
        Files{paths.length > 1 ? ` · ${paths.length}` : ""}
      </div>
      <div className="truncate text-sm">
        {basename(first)}
        {rest > 0 ? <span className="text-muted"> 等 {rest} 项</span> : null}
      </div>
    </div>
  );
};

export default FilesCard;
