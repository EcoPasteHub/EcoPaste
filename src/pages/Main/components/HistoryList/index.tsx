import { useFocusWithin, useKeyPress, useUpdateEffect } from "ahooks";
import { FloatButton, Modal } from "antd";
import { findIndex } from "es-toolkit/compat";
import { useContext, useEffect, useRef } from "react";
import Scrollbar from "@/components/Scrollbar";
import { LISTEN_KEY, PRESET_SHORTCUT } from "@/constants";
import { useHistoryList } from "@/hooks/useHistoryList";
import { useTauriListen } from "@/hooks/useTauriListen";
import { MainContext } from "../..";
import Item from "./components/Item";
import NoteModal, { type NoteModalRef } from "./components/NoteModal";

const HistoryList = () => {
  const { rootState } = useContext(MainContext);
  const noteModelRef = useRef<NoteModalRef>(null);
  const [deleteModal, contextHolder] = Modal.useModal();
  const scrollRef = useRef<HTMLDivElement>(null);
  const isFocusWithin = useFocusWithin(document.body);

  const { list, height, measureElement, scrollToIndex } =
    useHistoryList(scrollRef);

  const scrollToTop = () => {
    if (rootState.list.length === 0) return;

    scrollToIndex(0);

    rootState.activeId = rootState.list[0].id;
  };

  useTauriListen(LISTEN_KEY.ACTIVATE_BACK_TOP, scrollToTop);

  useUpdateEffect(() => {
    const { list } = rootState;

    if (list.length === 0) {
      rootState.activeId = void 0;
    } else {
      rootState.activeId ??= list[0].id;
    }
  }, [rootState.list.length]);

  useEffect(() => {
    const { list, activeId } = rootState;

    if (!activeId) return;

    const index = findIndex(list, { id: activeId });

    if (index < 0) return;

    scrollToIndex(index);
  }, [rootState.activeId]);

  useKeyPress(
    [
      "space",
      "enter",
      "backspace",
      "delete",
      "uparrow",
      "downarrow",
      "home",
      PRESET_SHORTCUT.FAVORITE,
    ],
    (event, key) => {
      event.preventDefault();

      if (key === "home") {
        return scrollToTop();
      }

      const { activeId, eventBus } = rootState;

      if (!activeId) return;

      switch (key) {
        // 空格预览
        case "space":
          return eventBus?.emit({
            action: LISTEN_KEY.CLIPBOARD_ITEM_PREVIEW,
            id: activeId,
          });
        // 回车粘贴
        case "enter":
          return eventBus?.emit({
            action: LISTEN_KEY.CLIPBOARD_ITEM_PASTE,
            id: activeId,
          });
        // 删除
        case "backspace":
        case "delete":
          return eventBus?.emit({
            action: LISTEN_KEY.CLIPBOARD_ITEM_DELETE,
            id: activeId,
          });
        // 选中上一个
        case "uparrow":
          return eventBus?.emit({
            action: LISTEN_KEY.CLIPBOARD_ITEM_SELECT_PREV,
            id: activeId,
          });
        // 选中下一个
        case "downarrow":
          return eventBus?.emit({
            action: LISTEN_KEY.CLIPBOARD_ITEM_SELECT_NEXT,
            id: activeId,
          });
        // 收藏和取消收藏
        case PRESET_SHORTCUT.FAVORITE:
          return eventBus?.emit({
            action: LISTEN_KEY.CLIPBOARD_ITEM_FAVORITE,
            id: activeId,
          });
      }
    },
    {
      events: isFocusWithin ? [] : ["keydown"],
    },
  );

  return (
    <>
      <Scrollbar className="flex-1" offset={3} ref={scrollRef}>
        <div className="relative" data-tauri-drag-region style={{ height }}>
          {list.map((item) => {
            const { key, start, index } = item;

            const data = rootState.list[index];

            if (!data) return null;

            return (
              <div
                className="absolute w-full"
                data-index={index}
                key={key}
                ref={measureElement}
                style={{ top: start }}
              >
                <Item
                  data={data}
                  deleteModal={deleteModal}
                  handleNote={() => noteModelRef.current?.open(data.id)}
                  index={index}
                />
              </div>
            );
          })}
        </div>
      </Scrollbar>

      <NoteModal ref={noteModelRef} />

      <FloatButton.BackTop
        duration={0}
        onClick={scrollToTop}
        target={() => scrollRef.current!}
      />

      {contextHolder}
    </>
  );
};

export default HistoryList;
