import { useVirtualizer } from "@tanstack/react-virtual";
import { copyFile, exists, remove } from "@tauri-apps/plugin-fs";
import { isString, last } from "es-toolkit";
import { unionBy } from "es-toolkit/compat";
import type { RefObject } from "react";
import { getDefaultSaveImagePath } from "tauri-plugin-clipboard-x-api";
import { MainContext } from "@/pages/Main";

export const useHistoryList = (scrollRef: RefObject<HTMLDivElement>) => {
  const { rootState } = useContext(MainContext);
  const scrollPos = useScroll(scrollRef);
  const state = useReactive({
    loading: false,
    noMore: false,
    page: 1,
    size: 20,
  });

  const fetchData = async () => {
    try {
      if (state.loading) return;

      state.loading = true;

      const { page } = state;

      const list = await selectHistory((qb) => {
        const { size } = state;
        const { group, search } = rootState;
        const isFavoriteGroup = group === "favorite";
        const isNormalGroup = group !== "all" && !isFavoriteGroup;

        return qb
          .$if(isFavoriteGroup, (eb) => eb.where("favorite", "=", true))
          .$if(isNormalGroup, (eb) => eb.where("group", "=", group))
          .$if(!isBlank(search), (eb) => {
            return eb.where((eb) => {
              return eb.or([
                eb("search", "like", eb.val(`%${search}%`)),
                eb("note", "like", eb.val(`%${search}%`)),
              ]);
            });
          })
          .offset((page - 1) * size)
          .limit(size)
          .orderBy("createTime", "desc");
      });

      for (const item of list) {
        const { type, value } = item;

        if (!isString(value)) continue;

        if (type === "image") {
          const oldPath = join(getSaveImagePath(), value);
          const newPath = join(await getDefaultSaveImagePath(), value);

          if (await exists(oldPath)) {
            await copyFile(oldPath, newPath);

            remove(oldPath);
          }

          item.value = newPath;
        }

        if (type === "files") {
          item.value = JSON.parse(value);
        }
      }

      state.noMore = list.length === 0;

      if (page === 1) {
        rootState.list = list;

        if (state.noMore) return;

        return virtualizer.scrollToIndex(0);
      }

      rootState.list = unionBy(rootState.list, list, "id");
    } finally {
      state.loading = false;
    }
  };

  const reload = () => {
    state.page = 1;
    state.noMore = false;

    return fetchData();
  };

  const loadMore = () => {
    if (state.noMore) return;

    state.page += 1;

    fetchData();
  };

  useTauriListen(LISTEN_KEY.REFRESH_CLIPBOARD_LIST, reload);

  useAsyncEffect(async () => {
    await reload();

    rootState.activeId = rootState.list[0]?.id;
  }, [rootState.group, rootState.search]);

  useEffect(() => {
    if (!scrollRef.current || !scrollPos) return;

    const { top } = scrollPos;

    if (rootState.list.length > 20 && top <= 0) {
      reload();
    } else {
      const { clientHeight, scrollHeight } = scrollRef.current;

      if (top + clientHeight >= scrollHeight - 50) {
        loadMore();
      }
    }
  }, [rootState.list.length, scrollPos]);

  const virtualizer = useVirtualizer({
    count: state.noMore ? rootState.list.length : rootState.list.length + 1,
    estimateSize: () => 54,
    gap: 12,
    getScrollElement: () => scrollRef.current,
    overscan: 5,
  });

  const list = virtualizer.getVirtualItems();

  const height = useCreation(() => {
    const lastItem = last(list);

    if (!lastItem) return 0;

    return lastItem.end - lastItem.size - 12;
  }, [list]);

  return {
    ...virtualizer,
    height,
    list,
  };
};
