import dayjs from "dayjs";
import type { FC } from "react";
import AssetImage from "@/components/AssetImage";
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
  const { kind, subKind, sourceAppIconPath, sourceAppName } = item;

  return (
    <div className="b b-border-secondary flex flex-col gap-1 rounded-2 p-2">
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
