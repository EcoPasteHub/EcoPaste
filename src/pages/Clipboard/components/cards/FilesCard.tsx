import type { FC } from "react";
import AssetImage from "@/components/AssetImage";
import type { ClipboardItem } from "@/types/clipboard";

/**
 * 文件类卡片：`fileEntries` 由 Rust 命令层预处理（最多前 3 条，含 icon / 文件名 / 类型判定），
 * 单文件且为图片时直接渲染图片预览，其它走文件列表。
 */
const FilesCard: FC<ClipboardItem> = (props) => {
  const entries = props.fileEntries ?? [];
  const [first] = entries;

  if (entries.length === 1 && first?.isImage) {
    return <AssetImage className="max-h-21" src={first.path} />;
  }

  return (
    <div className="flex flex-col gap-1">
      {entries.map((entry) => (
        <div
          className="flex items-center gap-1 truncate"
          key={entry.path}
          title={entry.path}
        >
          <AssetImage className="size-5" src={entry.iconPath} />

          <span className="truncate">{entry.name}</span>
        </div>
      ))}
    </div>
  );
};

export default FilesCard;
