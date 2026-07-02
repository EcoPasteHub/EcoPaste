import type { DragEvent, FC, MouseEvent, PointerEvent, Ref } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { popupClipboardItemMenu, startDragClipboardItem } from "@/commands";
import AssetImage from "@/components/AssetImage";
import KeyHint from "@/components/KeyHint";
import type { ItemActionLabels } from "@/constants/itemActions";
import type { ClipboardAction, ClipboardItem } from "@/types/clipboard";
import type { ItemAction } from "@/types/settings";
import { cn } from "@/utils/cn";
import ClipboardQuickActions from "./ClipboardQuickActions";
import FilesCard from "./FilesCard";
import ImageCard from "./ImageCard";
import NoteContentSwitcher from "./NoteContentSwitcher";
import TextCard from "./TextCard";

const TYPE_ICON_CLASS_BY_KEY: Record<string, string> = {
  color: "i-lucide:palette",
  email: "i-lucide:mail",
  files: "i-lucide:files",
  html: "i-lucide:code-xml",
  image: "i-lucide:image",
  path: "i-lucide:folder",
  rtf: "i-lucide:file-type",
  text: "i-lucide:text",
  url: "i-lucide:link",
};

interface ClipboardCardProps {
  bottomSheet?: boolean;
  className?: string;
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
  /**
   * MOD 键按下时，URL / Email 文本以链接态展示。
   */
  isLinkActive?: boolean;
  /**
   * 点击 URL / Email 文本时打开外部链接。
   */
  onOpenLink?: () => void;
  onPointerEnter?: (event: PointerEvent<HTMLDivElement>) => void;
  onPointerLeave?: () => void;
  onPointerMove?: (event: PointerEvent<HTMLDivElement>) => void;
  onMouseDown?: (event: MouseEvent<HTMLDivElement>) => void;
  onAuxClick?: (event: MouseEvent<HTMLDivElement>) => void;
  onDoubleClick?: (event: MouseEvent<HTMLDivElement>) => void;
  availableActions?: ClipboardAction[];
  quickActions?: ItemAction[];
  quickActionLabels?: ItemActionLabels;
  onQuickAction?: (action: ItemAction) => Promise<void> | void;
  showOriginalOnHover?: boolean;
  rootRef?: Ref<HTMLDivElement>;
}

/**
 * 按 `kind` 分发到具体卡片组件，统一外层 padding / 时间戳 / 来源应用图标。
 * `isSelected` 为 true 时高亮背景与边框；指针事件由列表注入用于 hover preview；
 * 右键根节点弹出 Rust 端原生菜单（避免 tauri-apps/tauri#9470 的 muda use-after-free），
 * 点击菜单项后由列表层订阅 `clipboard://menu-action` 派发到实际处理逻辑。
 */
const ClipboardCard: FC<ClipboardCardProps> = (props) => {
  const {
    item,
    bottomSheet = false,
    className,
    isSelected,
    hintKey,
    onQuickPaste,
    isLinkActive,
    onOpenLink,
    onPointerEnter,
    onPointerLeave,
    onPointerMove,
    onMouseDown,
    onAuxClick,
    onDoubleClick,
    availableActions,
    quickActions = [],
    quickActionLabels,
    onQuickAction,
    showOriginalOnHover = true,
    rootRef,
  } = props;
  const { kind, subKind, sourceAppIconPath, sourceAppName } = item;
  const { t } = useTranslation("clipboard");
  const [hovered, setHovered] = useState(false);
  const typeKey = subKind ?? kind;
  const typeLabel = t(`types.${typeKey}`);
  const body = renderBody(item, isLinkActive, onOpenLink, bottomSheet);
  const showSensitiveIndicator = item.isSensitive && item.kind === "text";
  const showStatusIndicators = item.isPinned || showSensitiveIndicator;

  const handleDragStart = async (event: DragEvent) => {
    event.preventDefault();

    await startDragClipboardItem(item.id);
  };

  const handleContextMenu = async (event: MouseEvent) => {
    event.preventDefault();

    const actions = availableActions ?? item.availableActions ?? [];
    const { isFavorite, isPinned, note } = item;

    if (actions.length === 0) return;

    await popupClipboardItemMenu(
      item.id,
      [...actions],
      item.groupId,
      isFavorite,
      isPinned,
      Boolean(note),
    );
  };

  const handlePointerEnter = (event: PointerEvent<HTMLDivElement>) => {
    setHovered(true);
    onPointerEnter?.(event);
  };

  const handlePointerLeave = () => {
    setHovered(false);
    onPointerLeave?.();
  };

  return (
    <div
      aria-selected={isSelected}
      className={cn(
        "relative flex flex-col gap-1 overflow-hidden rounded-2 border border-ant-border-secondary p-2 transition-colors duration-150 ease-out motion-reduce:transition-none",
        {
          "bg-ant-container shadow-md hover:shadow-lg": bottomSheet,
          "border-ant-primary bg-ant-blue-1": isSelected,
          "border-ant-primary bg-ant-container": item.isPinned && !isSelected,
          "gap-3 rounded-2.5 p-4": bottomSheet,
          "gap-4": bottomSheet && item.kind === "image",
        },
        className,
      )}
      draggable
      onAuxClick={onAuxClick}
      onContextMenu={handleContextMenu}
      onDoubleClick={onDoubleClick}
      onDragStart={handleDragStart}
      onMouseDown={onMouseDown}
      onPointerEnter={handlePointerEnter}
      onPointerLeave={handlePointerLeave}
      onPointerMove={onPointerMove}
      ref={rootRef}
      role="option"
      tabIndex={-1}
    >
      <div className="flex items-center justify-between text-ant-secondary text-xs">
        <div
          className={cn("flex min-w-0 items-center overflow-hidden", {
            "gap-1": !bottomSheet,
            "gap-1.5": bottomSheet,
          })}
        >
          {bottomSheet
            ? null
            : renderSourceIcon(
                sourceAppIconPath,
                sourceAppName,
                hintKey,
                onQuickPaste,
                bottomSheet,
              )}

          {renderTypeLabel(typeKey, typeLabel, bottomSheet)}
        </div>

        <ClipboardQuickActions
          item={item}
          labels={quickActionLabels}
          onQuickAction={onQuickAction}
          quickActions={quickActions}
          visible={hovered}
        />
      </div>

      {item.note ? (
        <div
          className={cn({
            "flex min-h-0 w-full flex-1 items-center justify-center px-2 pb-7 text-center":
              bottomSheet,
          })}
        >
          <NoteContentSwitcher
            note={item.note}
            showOriginal={showOriginalOnHover && hovered}
          >
            <div
              className={cn({
                "flex min-h-0 w-full flex-1 items-center justify-center px-2 pb-7 text-center":
                  bottomSheet,
              })}
            >
              {body}
            </div>
          </NoteContentSwitcher>
        </div>
      ) : (
        <div
          className={cn({
            "flex min-h-0 w-full flex-1 items-center justify-center px-2 pb-7 text-center":
              bottomSheet,
          })}
        >
          {body}
        </div>
      )}
      {showStatusIndicators
        ? renderStatusIndicators(
            item.isPinned,
            showSensitiveIndicator,
            bottomSheet,
          )
        : null}
      {bottomSheet
        ? renderSourceIcon(
            sourceAppIconPath,
            sourceAppName,
            hintKey,
            onQuickPaste,
            bottomSheet,
          )
        : null}
    </div>
  );
};

/**
 * 渲染卡片右下角的状态水印；仅表达状态，不参与交互。
 */
function renderStatusIndicators(
  isPinned: boolean,
  isSensitive: boolean,
  bottomSheet: boolean,
) {
  return (
    <div
      className={cn(
        "pointer-events-none absolute bottom-2 flex items-end gap-1 text-ant-quaternary",
        {
          "left-3": bottomSheet,
          "right-2": !bottomSheet,
        },
      )}
    >
      {isPinned ? (
        <i aria-hidden="true" className="i-ph:push-pin-bold size-5" />
      ) : null}
      {isSensitive ? (
        <i aria-hidden="true" className="i-lucide:key-round size-5" />
      ) : null}
    </div>
  );
}

function renderTypeLabel(
  typeKey: string,
  typeLabel: string,
  bottomSheet: boolean,
) {
  const iconClassName =
    TYPE_ICON_CLASS_BY_KEY[typeKey] ?? TYPE_ICON_CLASS_BY_KEY.text;

  if (!bottomSheet) return <span className="truncate">{typeLabel}</span>;

  return (
    <span className="inline-flex max-w-full items-center gap-1.5 truncate rounded-full border border-ant-border-secondary bg-ant-fill-secondary px-2.5 py-1 font-medium text-ant-text text-sm shadow-sm">
      <i aria-hidden="true" className={cn("shrink-0 text-lg", iconClassName)} />
      <span className="truncate">{typeLabel}</span>
    </span>
  );
}

function renderSourceIcon(
  sourceAppIconPath: string | null | undefined,
  sourceAppName: string | undefined,
  hintKey: string | undefined,
  onQuickPaste: (() => void) | undefined,
  bottomSheet: boolean,
) {
  const icon = (
    <AssetImage
      alt={sourceAppName ?? ""}
      className={cn({
        "size-4": !bottomSheet,
        "size-7 rounded-1.5": bottomSheet,
      })}
      src={sourceAppIconPath}
    />
  );

  const content = hintKey ? (
    <KeyHint hintKey={hintKey} onKeyPress={onQuickPaste}>
      {icon}
    </KeyHint>
  ) : (
    icon
  );

  if (!bottomSheet) return content;

  return (
    <div className="absolute right-2 bottom-2 flex size-9 items-center justify-center rounded-2 bg-ant-container shadow-md">
      {content}
    </div>
  );
}

const renderBody = (
  item: ClipboardItem,
  isLinkActive?: boolean,
  onOpenLink?: () => void,
  bottomSheet?: boolean,
) => {
  if (item.kind === "image") {
    return <ImageCard {...item} bottomSheet={bottomSheet} />;
  }

  if (item.kind === "files")
    return <FilesCard {...item} bottomSheet={bottomSheet} />;

  return (
    <TextCard
      {...item}
      bottomSheet={bottomSheet}
      isLinkActive={isLinkActive}
      onOpenLink={onOpenLink}
    />
  );
};

export default ClipboardCard;
