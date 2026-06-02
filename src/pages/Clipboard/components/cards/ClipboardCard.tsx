import type { DragEvent, FC, MouseEvent } from "react";
import { popupClipboardItemMenu, startDragClipboardItem } from "@/commands";
import AssetImage from "@/components/AssetImage";
import KeyHint from "@/components/KeyHint";
import type { ClipboardItem } from "@/types/clipboard";
import { cn } from "@/utils/cn";
import FilesCard from "./FilesCard";
import ImageCard from "./ImageCard";
import TextCard from "./TextCard";

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
}

/**
 * 按 `kind` 分发到具体卡片组件，统一外层 padding / 时间戳 / 来源应用图标。
 * `isSelected` 为 true 时高亮背景与边框；`onMouseEnter` 由列表注入用于鼠标悬停选中；
 * 右键根节点弹出 Rust 端原生菜单（避免 tauri-apps/tauri#9470 的 muda use-after-free），
 * 点击菜单项后由列表层订阅 `clipboard://menu-action` 派发到实际处理逻辑。
 */
const ClipboardCard: FC<ClipboardCardProps> = (props) => {
  const { item, isSelected, hintKey, onQuickPaste, onMouseEnter } = props;
  const { kind, subKind, sourceAppIconPath, sourceAppName } = item;

  // text 类型 drag-out 暂未支持（drag crate 的 Data 在 Windows 是 dummy 实现），只挂在
  // files / image 卡片上；text 卡片维持原有选中文字行为，避免误触。
  const draggable = kind === "files" || kind === "image";

  // Windows 上 DoDragDrop 必须在浏览器 dragstart 上下文里调用，否则 QueryContinueDrag
  // 立刻返回 DRAGDROP_S_CANCEL；参考 drag-rs/examples/tauri/index.html。
  const handleDragStart = async (event: DragEvent) => {
    event.preventDefault();

    try {
      await startDragClipboardItem(item.id);
    } catch {
      // 错误 toast 已在 commands/index.ts 内统一处理。
    }
  };

  const handleContextMenu = async (event: MouseEvent) => {
    event.preventDefault();

    const { availableActions = [], isFavorite } = item;

    if (availableActions.length === 0) return;

    await popupClipboardItemMenu(item.id, [...availableActions], isFavorite);
  };

  return (
    <div
      aria-selected={isSelected}
      className={cn("b b-border-secondary flex flex-col gap-1 rounded-2 p-2", {
        "b-primary bg-blue-1": isSelected,
      })}
      draggable={draggable}
      onContextMenu={handleContextMenu}
      onDragStart={draggable ? handleDragStart : void 0}
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

        <span>{item.displayCreatedAt ?? item.createdAt}</span>
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

export default ClipboardCard;
