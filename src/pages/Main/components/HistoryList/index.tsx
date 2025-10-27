import { useUpdateEffect } from "ahooks";
import { FloatButton, Modal } from "antd";
import clsx from "clsx";
import { findIndex } from "es-toolkit/compat";
import { useContext, useEffect, useRef } from "react";
import { Virtuoso, type VirtuosoHandle } from "react-virtuoso";
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
  const virtuosoRef = useRef<VirtuosoHandle>(null);
  const scrollerRef = useRef<HTMLDivElement>(null);

  const scrollToIndex = (index: number) => {
    return virtuosoRef.current?.scrollIntoView({ index });
  };

  const scrollToTop = () => {
    if (rootState.list.length === 0) return;

    scrollToIndex(0);

    rootState.activeId = rootState.list[0].id;
  };

  useKeyboard({ scrollToTop });

  const { reload, loadMore } = useHistoryList({ scrollToTop });

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
      <Scrollbar className="flex-1" offsetX={3} ref={scrollerRef}>
        <Virtuoso
          atTopStateChange={(atTop) => {
            if (!atTop || rootState.list.length <= 20) return;

            reload();
          }}
          computeItemKey={(_, item) => item.id}
          customScrollParent={scrollerRef.current ?? void 0}
          data={rootState.list}
          endReached={loadMore}
          itemContent={(index, data) => {
            return (
              <div className={clsx({ "pt-3": index !== 0 })}>
                <Item
                  data={data}
                  deleteModal={deleteModal}
                  handleNote={() => noteModelRef.current?.open(data.id)}
                  index={index}
                />
              </div>
            );
          }}
          ref={virtuosoRef}
        />
      </Scrollbar>

      <NoteModal ref={noteModelRef} />

      <FloatButton.BackTop
        duration={0}
        onClick={scrollToTop}
        target={() => scrollerRef.current!}
      />

      {contextHolder}
    </>
  );
};

export default HistoryList;
