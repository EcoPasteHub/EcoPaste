import { useUpdateEffect } from "ahooks";
import { FloatButton, Modal } from "antd";
import { findIndex } from "es-toolkit/compat";
import { useContext, useEffect, useRef } from "react";
import Scrollbar from "@/components/Scrollbar";
import { LISTEN_KEY } from "@/constants";
import { useHistoryList } from "@/hooks/useHistoryList";
import { useKeyboard } from "@/hooks/useKeyboard";
import { useTauriListen } from "@/hooks/useTauriListen";
import { MainContext } from "../..";
import Item from "./components/Item";
import NoteModal, { type NoteModalRef } from "./components/NoteModal";

const HistoryList = () => {
  const { rootState } = useContext(MainContext);
  const noteModelRef = useRef<NoteModalRef>(null);
  const [deleteModal, contextHolder] = Modal.useModal();
  const scrollRef = useRef<HTMLDivElement>(null);

  const { list, height, measureElement, scrollToIndex } =
    useHistoryList(scrollRef);

  const scrollToTop = () => {
    if (rootState.list.length === 0) return;

    scrollToIndex(0);

    rootState.activeId = rootState.list[0].id;
  };

  useKeyboard({ scrollToTop });

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
