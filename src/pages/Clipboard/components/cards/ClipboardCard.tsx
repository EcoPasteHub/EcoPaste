import { useState } from "react";

import type { ClipboardItem } from "@/types/clipboard";
import { cn } from "@/utils/cn";

import type { AppMeta } from "../../hooks/useClipboardApps";
import type { ClipboardActions } from "../../hooks/useClipboardItems";
import FilesCard from "./FilesCard";
import ImageCard from "./ImageCard";
import TextCard from "./TextCard";

const formatTime = (iso: string): string => {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "";
  const diff = Date.now() - t;
  if (diff < 60_000) return "刚刚";
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} 分钟前`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} 小时前`;
  return iso.slice(0, 10);
};

const renderBody = (item: ClipboardItem, keyword: string) => {
  switch (item.kind) {
    case "image":
      return <ImageCard item={item} />;
    case "files":
      return <FilesCard item={item} />;
    default:
      return <TextCard item={item} keyword={keyword} />;
  }
};

/**
 * 卡片操作区里的小尺寸文字按钮：HeroUI 主题色 token，hover 时变深。
 */
const ActionButton = ({
  onClick,
  active = false,
  danger = false,
  children,
}: {
  onClick: () => void;
  active?: boolean;
  danger?: boolean;
  children: React.ReactNode;
}) => {
  return (
    <button
      className={cn("rounded px-1.5 py-0.5 text-xs", {
        "c-error hover:c-error-hover": danger,
        "c-link": !danger && active,
        "c-text-tertiary hover:c-text": !danger && !active,
      })}
      onClick={(e) => {
        e.stopPropagation();
        onClick();
      }}
      type="button"
    >
      {children}
    </button>
  );
};

const ClipboardCard = ({
  item,
  actions,
  app,
  isSelected = false,
  keyword = "",
}: {
  item: ClipboardItem;
  actions: ClipboardActions;
  app?: AppMeta;
  isSelected?: boolean;
  keyword?: string;
}) => {
  const [noteOpen, setNoteOpen] = useState(false);
  const [draft, setDraft] = useState(item.note ?? "");

  const openNote = () => {
    setDraft(item.note ?? "");
    setNoteOpen(true);
  };

  const saveNote = () => {
    actions.updateNote(item.id, draft);
    setNoteOpen(false);
  };

  return (
    <div
      className={cn("group b-split relative border-b px-3 py-2", {
        "c-primary-text bg-primary-bg": isSelected,
      })}
    >
      {renderBody(item, keyword)}

      {item.note ? (
        <div className="c-text-tertiary mt-1 truncate text-xs italic">
          备注：{item.note}
        </div>
      ) : null}

      <div className="c-text-tertiary mt-1 flex items-center justify-between text-xs">
        <span className="flex items-center gap-1.5">
          {app ? (
            app.iconSrc ? (
              <img
                alt=""
                className="size-3.5 shrink-0 rounded-sm"
                src={app.iconSrc}
                title={app.name}
              />
            ) : (
              <span
                className="size-3.5 shrink-0 rounded-sm bg-bg-elevated"
                title={app.name}
              />
            )
          ) : null}
          <span>{formatTime(item.createdAt)}</span>
        </span>
        <span className="flex items-center gap-2">
          {item.isPinned ? <span title="置顶">●</span> : null}
          {item.isFavorite ? <span title="收藏">★</span> : null}
        </span>
      </div>

      {/* hover 时显形：避免每条都堆满按钮，鼠标过来再露出 */}
      <div className="absolute top-1 right-2 hidden gap-1 rounded bg-bg-container shadow-sm group-hover:flex">
        <ActionButton onClick={() => actions.paste(item.id)}>粘贴</ActionButton>
        <ActionButton onClick={() => actions.copy(item.id)}>复制</ActionButton>
        <ActionButton
          active={item.isFavorite}
          onClick={() => actions.toggleFavorite(item.id)}
        >
          {item.isFavorite ? "取消收藏" : "收藏"}
        </ActionButton>
        <ActionButton onClick={openNote}>备注</ActionButton>
        <ActionButton danger onClick={() => actions.remove(item.id)}>
          删除
        </ActionButton>
      </div>

      {noteOpen ? (
        // biome-ignore lint/a11y/noStaticElementInteractions: 内层 stop-propagation 用于隔离父级事件，没有实际语义角色
        <div
          className="mt-2 flex flex-col gap-1"
          onClick={(e) => e.stopPropagation()}
          onKeyDown={(e) => e.stopPropagation()}
        >
          <textarea
            className="b-split rounded border bg-bg-elevated px-2 py-1 text-xs"
            onChange={(e) => setDraft(e.target.value)}
            placeholder="备注..."
            ref={(el) => el?.focus()}
            rows={2}
            value={draft}
          />
          <div className="flex justify-end gap-1">
            <ActionButton onClick={() => setNoteOpen(false)}>取消</ActionButton>
            <ActionButton onClick={saveNote}>保存</ActionButton>
          </div>
        </div>
      ) : null}
    </div>
  );
};

export default ClipboardCard;
