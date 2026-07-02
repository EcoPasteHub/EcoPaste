import type { FC } from "react";
import { useSnapshot } from "valtio";
import AssetImage from "@/components/AssetImage";
import Highlight from "@/components/Highlight";
import { clipboardViewState } from "@/stores/clipboardView";
import type { ClipboardItem, FileEntry } from "@/types/clipboard";
import { cn } from "@/utils/cn";
import ImageCard from "./ImageCard";

interface FilesCardProps extends ClipboardItem {
  bottomSheet?: boolean;
}

/**
 * 文件类卡片：`fileEntries` 与 `filesPreviewKind` 均由 Rust 命令层预处理，
 * 前端只按 kind 分支渲染：`imagePreview` 走单图预览，`list` 走文件列表；
 * 路径已删除（`exists = false`）的条目一律走 list 并对文件名划删除线。
 */
const FilesCard: FC<FilesCardProps> = (props) => {
  const entries = props.fileEntries ?? [];
  const { bottomSheet = false } = props;
  const { keyword } = useSnapshot(clipboardViewState);

  if (props.filesPreviewKind === "imagePreview") {
    const [first] = entries;
    return (
      <ImageCard
        {...props}
        bottomSheet={bottomSheet}
        imageThumbnailPath={first?.path}
      />
    );
  }

  return (
    <div
      className={cn("flex flex-col", {
        "gap-1": !bottomSheet,
        "items-center gap-2 font-medium text-lg": bottomSheet,
      })}
    >
      {entries.map((entry) => {
        return (
          <FileRow
            bottomSheet={bottomSheet}
            entry={entry}
            key={entry.path}
            keyword={keyword}
          />
        );
      })}
    </div>
  );
};

interface FileRowProps {
  bottomSheet?: boolean;
  entry: FileEntry;
  keyword: string;
}

const FileRow: FC<FileRowProps> = (props) => {
  const { bottomSheet = false, entry, keyword } = props;

  return (
    <div
      className={cn("flex items-center truncate", {
        "gap-1": !bottomSheet,
        "gap-2": bottomSheet,
      })}
      title={entry.path}
    >
      <AssetImage
        className={cn({
          "size-5": !bottomSheet,
          "size-7": bottomSheet,
        })}
        src={entry.iconPath}
      />

      <Highlight
        className={cn("truncate", { "line-through": !entry.exists })}
        keyword={keyword}
        text={entry.name}
      />
    </div>
  );
};

export default FilesCard;
