import { startDrag } from "@crabnebula/tauri-plugin-drag";
import { resolveResource } from "@tauri-apps/api/path";
import { openPath } from "@tauri-apps/plugin-opener";
import { Flex, message } from "antd";
import type { HookAPI } from "antd/es/modal/useModal";
import clsx from "clsx";
import type { DragEvent, FC } from "react";
import { useSnapshot } from "valtio";
import UnoIcon from "@/components/UnoIcon";
import { MainContext } from "@/pages/Main";
import type { DatabaseSchemaHistory } from "@/types/database";
import Files from "../Files";
import Header from "../Header";
import HTML from "../HTML";
import Image from "../Image";
import RTF from "../RTF";
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

  const { handleContextMenu, ...rest } = useContextMenu(props);

  const handlePreview = () => {
    if (type !== "image") return;

    openPath(value);
  };

  const handlePaste = () => {
    pasteToClipboard(data, content.pastePlain);
  };

  const handleNext = () => {
    const nextIndex = index + 1;

    if (nextIndex >= rootState.list.length) return;

    rootState.activeId = rootState.list[nextIndex].id;
  };

  const handlePrev = () => {
    const nextIndex = index - 1;

    if (nextIndex < 0) return;

    rootState.activeId = rootState.list[nextIndex].id;
  };

  rootState.eventBus?.useSubscription((payload) => {
    if (payload.id !== id) return;

    const { handleDelete, handleFavorite } = rest;

    switch (payload.action) {
      case LISTEN_KEY.CLIPBOARD_ITEM_PREVIEW:
        return handlePreview();
      case LISTEN_KEY.CLIPBOARD_ITEM_PASTE:
        return handlePaste();
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

  const handleClick = (type: typeof content.autoPaste) => {
    rootState.activeId = id;

    if (content.autoPaste !== type) return;

    handlePaste();
  };

  const handleDragStart = async (event: DragEvent) => {
    event.preventDefault();

    const icon = await resolveResource("assets/drag-icon.png");

    if (type === "image") {
      return startDrag({ icon: value, item: [value] });
    }

    if (type === "files") {
      return startDrag({ icon, item: value });
    }

    return message.warning("暂不支持拖拽文本");
  };

  const renderContent = () => {
    switch (type) {
      case "text":
        return <Text {...data} />;
      case "rtf":
        return <RTF {...data} />;
      case "html":
        return <HTML {...data} />;
      case "image":
        return <Image {...data} />;
      case "files":
        return <Files {...data} />;
    }
  };

  return (
    <Flex
      className={clsx(
        "group antd-input! b-color-2 mx-3 max-h-30 rounded-md p-1.5",
        {
          "antd-input-focus!": rootState.activeId === id,
        },
      )}
      draggable
      gap={4}
      onClick={() => handleClick("single")}
      onContextMenu={handleContextMenu}
      onDoubleClick={() => handleClick("double")}
      onDragStart={handleDragStart}
      vertical
    >
      <Header {...rest} data={data} handleNote={handleNote} />

      <div className="relative flex-1 select-auto overflow-hidden break-words children:transition">
        <div
          className={clsx(
            "pointer-events-none absolute inset-0 line-clamp-4 opacity-0",
            {
              "group-hover:opacity-0": content.showOriginalContent,
              "opacity-100": note,
            },
          )}
        >
          <UnoIcon
            className="mr-0.5 translate-y-0.5"
            name="i-hugeicons:task-edit-01"
          />

          {note}
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
    </Flex>
  );
};

export default Item;
