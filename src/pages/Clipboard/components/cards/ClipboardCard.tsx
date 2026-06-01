import { convertFileSrc } from "@tauri-apps/api/core";
import dayjs from "dayjs";
import type { FC } from "react";
import type { ClipboardItem } from "@/types/clipboard";
import FilesCard from "./FilesCard";
import ImageCard from "./ImageCard";
import TextCard from "./TextCard";

interface ClipboardCardProps {
  item: ClipboardItem;
}

/**
 * 按 `kind` 分发到具体卡片组件，统一外层 padding / 时间戳 / 来源应用图标。
 */
const ClipboardCard: FC<ClipboardCardProps> = (props) => {
  const { item } = props;

  const iconSrc = item.sourceAppIconPath
    ? convertFileSrc(item.sourceAppIconPath)
    : null;
  const label = item.sourceAppName ?? item.kind.toUpperCase();

  return (
    <div className="flex flex-col gap-1.5 rounded-2 border border-gray-200/60 bg-elevated px-2.5 py-2 transition-colors hover:bg-fill-tertiary dark:border-gray-700/40">
      <div className="flex items-center justify-between gap-2 text-gray-400 text-xs">
        <div className="flex items-center gap-1.5 overflow-hidden">
          {iconSrc ? (
            <img
              alt={label}
              className="size-3.5 shrink-0 rounded-0.5"
              src={iconSrc}
            />
          ) : null}
          <span className="truncate">{label}</span>
        </div>
        <span className="shrink-0">{formatTime(item.createdAt)}</span>
      </div>

      {renderBody(item)}
    </div>
  );
};

const renderBody = (item: ClipboardItem) => {
  if (item.kind === "image") return <ImageCard {...item} />;

  if (item.kind === "files") return <FilesCard {...item} />;

  return <TextCard {...item} />;
};

const formatTime = (iso: string) => {
  const value = dayjs(iso);

  if (!value.isValid()) return iso;

  const now = dayjs();

  if (value.isSame(now, "day")) return value.format("HH:mm:ss");

  if (value.isSame(now, "year")) return value.format("MM-DD HH:mm");

  return value.format("YYYY-MM-DD HH:mm");
};

export default ClipboardCard;
