import { openPath } from "@tauri-apps/plugin-opener";
import { Flex } from "antd";
import type { HookAPI } from "antd/es/modal/useModal";
import clsx from "clsx";
import { type FC, useContext, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Marker } from "react-mark.js";
import { useSnapshot } from "valtio";
import SafeHtml from "@/components/SafeHtml";
import UnoIcon from "@/components/UnoIcon";
import { LISTEN_KEY } from "@/constants";
import { useContextMenu } from "@/hooks/useContextMenu";
import { MainContext } from "@/pages/Main";
import { pasteToClipboard } from "@/plugins/clipboard";
import { clipboardStore } from "@/stores/clipboard";
import type { DatabaseSchemaHistory } from "@/types/database";
import Files from "../Files";
import Header from "../Header";
import Image from "../Image";
import Rtf from "../Rtf";
import Text from "../Text";

export interface ItemProps {
  index: number;
  data: DatabaseSchemaHistory;
  deleteModal: HookAPI;
  handleNote: () => void;
}

const Item: FC<ItemProps> = (props) => {
  const { index, data, handleNote } = props;
  const { id, type, note, value } = data;
  const { rootState } = useContext(MainContext);
  const { content } = useSnapshot(clipboardStore);
  const { t } = useTranslation();
  const expanded = rootState.expandedIds.includes(id);
  const [isOverflow, setIsOverflow] = useState(expanded);
  const contentRef = useRef<HTMLDivElement | HTMLImageElement>(null);

  // 检查内容是否溢出（依赖 expanded 以便收起时重新检测）
  useEffect(() => {
    checkOverflow();
  }, [
    content.displayLines,
    content.imageDisplayHeight,
    value,
    type,
    rootState.search,
    expanded,
  ]);

  const checkOverflow = () => {
    // 展开状态：已展开说明之前检测过溢出，保持按钮显示
    if (expanded) {
      setIsOverflow(true);
      return;
    }

    if (!contentRef.current) {
      setIsOverflow(false);
      return;
    }

    const element = contentRef.current;

    if (element instanceof HTMLImageElement) {
      // 图片：超过设定高度 且 缩放后的渲染宽度未布满容器时显示按钮
      const maxH = content.imageDisplayHeight || 100;
      const containerWidth =
        element.parentElement?.clientWidth || element.clientWidth;
      // 计算 maxHeight 约束下的渲染宽度（保持宽高比）
      const renderedWidth =
        element.naturalHeight > 0
          ? element.naturalWidth * (maxH / element.naturalHeight)
          : element.naturalWidth;
      setIsOverflow(
        element.naturalHeight > maxH && renderedWidth < containerWidth,
      );
    } else {
      // 文本：检查 scrollHeight 是否大于 clientHeight
      setIsOverflow(element.scrollHeight > element.clientHeight + 1);
    }
  };

  const handlePreview = () => {
    if (type !== "image") return;

    openPath(value);
  };

  const handleNext = () => {
    const { list } = rootState;

    const nextItem = list[index + 1] ?? list[index - 1];

    rootState.activeId = nextItem?.id;
  };

  const handlePrev = () => {
    if (index === 0) return;

    rootState.activeId = rootState.list[index - 1].id;
  };

  rootState.eventBus?.useSubscription((payload) => {
    if (payload.id !== id) return;

    const { handleDelete, handleFavorite } = rest;

    switch (payload.action) {
      case LISTEN_KEY.CLIPBOARD_ITEM_PREVIEW:
        return handlePreview();
      case LISTEN_KEY.CLIPBOARD_ITEM_PASTE:
        return pasteToClipboard(data);
      case LISTEN_KEY.CLIPBOARD_ITEM_DELETE:
        return handleDelete();
      case LISTEN_KEY.CLIPBOARD_ITEM_SELECT_PREV:
        return handlePrev();
      case LISTEN_KEY.CLIPBOARD_ITEM_SELECT_NEXT:
        return handleNext();
      case LISTEN_KEY.CLIPBOARD_ITEM_FAVORITE:
        return handleFavorite();
    }
  });

  const { handleContextMenu, ...rest } = useContextMenu({
    ...props,
    handleNext,
  });

  const handleClick = (type: typeof content.autoPaste) => {
    // 检查是否有选中文本，如果有则不触发粘贴
    const selection = window.getSelection();
    if (selection && selection.toString().length > 0) {
      return;
    }

    rootState.activeId = id;

    if (content.autoPaste !== type) return;

    pasteToClipboard(data);
  };

  const handleToggleExpand = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (expanded) {
      rootState.expandedIds = rootState.expandedIds.filter(
        (eid: string) => eid !== id,
      );
    } else {
      rootState.expandedIds = [...rootState.expandedIds, id];
    }
  };

  const renderContent = () => {
    switch (type) {
      case "text":
        return <Text ref={contentRef as any} {...data} expanded={expanded} />;
      case "rtf":
        return (
          <Rtf
            ref={contentRef as any}
            {...data}
            expanded={expanded}
            onLoad={checkOverflow}
          />
        );
      case "html":
        return (
          <SafeHtml ref={contentRef as any} {...data} expanded={expanded} />
        );
      case "image":
        return (
          <Image
            ref={contentRef as any}
            {...data}
            expanded={expanded}
            onLoad={checkOverflow}
          />
        );
      case "files":
        return <Files {...data} />;
    }
  };

  return (
    <Flex
      className={clsx(
        "group b hover:b-primary-5 b-color-2 mx-3 rounded-md p-1.5 transition",
        {
          "b-primary bg-primary-1": rootState.activeId === id,
        },
      )}
      gap={4}
      onClick={() => handleClick("single")}
      onContextMenu={handleContextMenu}
      onDoubleClick={() => handleClick("double")}
      vertical
    >
      <Header {...rest} data={data} handleNote={handleNote} />

      <div className="relative flex-1 select-auto overflow-hidden break-words children:transition">
        <div
          className={clsx(
            "pointer-events-none absolute inset-0 children:inline opacity-0",
            {
              "group-hover:opacity-0": content.showOriginalContent,
              "opacity-100": note,
            },
          )}
          style={{
            display: expanded ? "block" : "-webkit-box",
            overflow: "hidden",
            WebkitBoxOrient: "vertical",
            WebkitLineClamp: expanded ? "none" : content.displayLines || 4,
          }}
        >
          <UnoIcon
            className="mr-0.5 translate-y-0.5"
            name="i-hugeicons:task-edit-01"
          />

          <Marker mark={rootState.search}>{note}</Marker>
        </div>

        <div
          className={clsx("h-full", {
            "group-hover:opacity-100": content.showOriginalContent,
            "opacity-0": note,
          })}
        >
          {renderContent()}
        </div>
      </div>

      {/* 展开/收起按钮 */}
      {(isOverflow || expanded) && (
        <div
          className="flex cursor-pointer items-center justify-center text-primary text-xs hover:text-primary-6"
          onClick={handleToggleExpand}
        >
          <UnoIcon
            className="mr-1"
            name={expanded ? "i-lucide:chevron-up" : "i-lucide:chevron-down"}
          />
          <span>
            {expanded
              ? t("preference.clipboard.content_settings.label.collapse")
              : t("preference.clipboard.content_settings.label.expand")}
          </span>
        </div>
      )}
    </Flex>
  );
};

export default Item;
