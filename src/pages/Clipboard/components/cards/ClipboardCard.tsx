import dayjs from "dayjs";
import type { FC } from "react";
import AssetImage from "@/components/AssetImage";
import KeyHint from "@/components/KeyHint";
import type { ClipboardItem } from "@/types/clipboard";
import { cn } from "@/utils/cn";
import FilesCard from "./FilesCard";
import ImageCard from "./ImageCard";
import TextCard from "./TextCard";
import { useContextMenu } from "./useContextMenu";

interface ClipboardCardProps {
  item: ClipboardItem;
  isSelected?: boolean;
  /**
   * 快捷键提示字符（"1"–"9" / "0"），存在时在 app 图标上叠加 KeyHint；
   * 按下修饰键（macOS ⌘ / Windows Ctrl）+ 该数字键触发快速粘贴。
   */
  hintKey?: string;
  /**
   * 快捷键触发时执行的粘贴操作，由父级列表注入。
   */
  onQuickPaste?: () => void;
  onMouseEnter?: () => void;
  /**
   * 删除成功后通知列表移除该项（删除命令不广播 clipboard://updated，靠本地更新）。
   */
  onRemoved: (id: string) => void;
  /**
   * 收藏切换成功后通知列表同步 isFavorite。
   */
  onFavoriteToggled: (id: string, isFavorite: boolean) => void;
  /**
   * 打开备注编辑弹窗（列表层单例）。
   */
  onEditNote: (item: ClipboardItem) => void;
}

/**
 * 按 `kind` 分发到具体卡片组件，统一外层 padding / 时间戳 / 来源应用图标。
 * `isSelected` 为 true 时高亮背景与边框；`onMouseEnter` 由列表注入用于鼠标悬停选中；
 * 右键根节点弹出 Tauri 原生菜单（见 useContextMenu）。
 */
const ClipboardCard: FC<ClipboardCardProps> = (props) => {
  const {
    item,
    isSelected,
    hintKey,
    onQuickPaste,
    onMouseEnter,
    onRemoved,
    onFavoriteToggled,
    onEditNote,
  } = props;
  const { kind, subKind, sourceAppIconPath, sourceAppName } = item;

  const handleContextMenu = useContextMenu({
    item,
    onEditNote,
    onFavoriteToggled,
    onRemoved,
  });

  return (
    <div
      aria-selected={isSelected}
      className={cn("b b-border-secondary flex flex-col gap-1 rounded-2 p-2", {
        "b-primary bg-blue-1": isSelected,
      })}
      onContextMenu={handleContextMenu}
      onMouseEnter={onMouseEnter}
      role="option"
      tabIndex={-1}
    >
      <div className="flex items-center justify-between text-secondary text-xs">
        <div className="flex items-center gap-1 overflow-hidden">
          {hintKey ? (
            <KeyHint hintKey={hintKey} onKeyPress={onQuickPaste}>
              <AssetImage
                alt={sourceAppName}
                className="size-4"
                src={sourceAppIconPath}
              />
            </KeyHint>
          ) : (
            <AssetImage
              alt={sourceAppName}
              className="size-4"
              src={sourceAppIconPath}
            />
          )}

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
