import dayjs from "dayjs";
import type { FC } from "react";
import AssetImage from "@/components/AssetImage";
import type { ClipboardItem } from "@/types/clipboard";
import { cn } from "@/utils/cn";
import FilesCard from "./FilesCard";
import ImageCard from "./ImageCard";
import TextCard from "./TextCard";

interface ClipboardCardProps {
  item: ClipboardItem;
  isSelected?: boolean;
  onMouseEnter?: () => void;
}

/**
 * 按 `kind` 分发到具体卡片组件，统一外层 padding / 时间戳 / 来源应用图标。
 * `isSelected` 为 true 时高亮背景与边框；`onMouseEnter` 由列表注入用于鼠标悬停选中。
 */
const ClipboardCard: FC<ClipboardCardProps> = (props) => {
  const { item, isSelected, onMouseEnter } = props;
  const { kind, subKind, sourceAppIconPath, sourceAppName } = item;

  return (
    <div
      aria-selected={isSelected}
      className={cn("b b-border-secondary flex flex-col gap-1 rounded-2 p-2", {
        "b-primary bg-blue-1": isSelected,
      })}
      onMouseEnter={onMouseEnter}
      role="option"
      tabIndex={-1}
    >
      <div className="flex items-center justify-between text-secondary text-xs">
        <div className="flex items-center gap-1 overflow-hidden">
          <AssetImage
            alt={sourceAppName}
            className="size-4"
            src={sourceAppIconPath}
          />

          <span className="truncate uppercase">{subKind ?? kind}</span>
        </div>

        <span>{formatTime(item.createdAt)}</span>
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
