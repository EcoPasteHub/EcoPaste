import type { FC } from "react";
import { useSnapshot } from "valtio";
import AssetImage from "@/components/AssetImage";
import Highlight from "@/components/Highlight";
import { clipboardViewState } from "@/stores/clipboardView";
import type { ClipboardItem } from "@/types/clipboard";

/**
 * 文件类卡片：`fileEntries` 与 `filesPreviewKind` 均由 Rust 命令层预处理，
 * 前端只按 kind 分支渲染：`imagePreview` 走单图预览，`list` 走文件列表。
 */
const FilesCard: FC<ClipboardItem> = (props) => {
  const entries = props.fileEntries ?? [];
  const { keyword } = useSnapshot(clipboardViewState);

  if (props.filesPreviewKind === "imagePreview") {
    const [first] = entries;
    return <AssetImage className="max-h-21" src={first?.path} />;
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

          <Highlight className="truncate" keyword={keyword} text={entry.name} />
        </div>
      ))}
    </div>
  );
};

export default FilesCard;
