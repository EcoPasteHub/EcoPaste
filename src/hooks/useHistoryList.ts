import { copyFile, exists, remove } from "@tauri-apps/plugin-fs";
import { useAsyncEffect, useReactive } from "ahooks";
import { isString } from "es-toolkit";
import { unionBy } from "es-toolkit/compat";
import { useContext } from "react";
import { getDefaultSaveImagePath } from "tauri-plugin-clipboard-x-api";
import { LISTEN_KEY } from "@/constants";
import { selectHistory } from "@/database/history";
import { MainContext } from "@/pages/Main";
import { isBlank } from "@/utils/is";
import { getSaveImagePath, join } from "@/utils/path";
import { useTauriListen } from "./useTauriListen";

interface Options {
  scrollToTop: () => void;
}

export const useHistoryList = (options: Options) => {
  const { scrollToTop } = options;
  const { rootState } = useContext(MainContext);
  const state = useReactive({
    loading: false,
    noMore: false,
    page: 1,
    size: 20,
    // 防止旧请求结果覆盖新搜索
    fetchId: 0,
    queryKey: "",
    pendingReload: false,
  });

  const getQueryKey = () => {
    const { group, search } = rootState;

    return JSON.stringify([group, search ?? ""]);
  };

  const fetchData = async (key: string, page: number) => {
    try {
      state.loading = true;

      const currentFetchId = ++state.fetchId;

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

      // 丢弃过期请求的结果，避免旧结果覆盖新搜索
      if (currentFetchId !== state.fetchId) return;
      if (key !== state.queryKey) return;

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

        return scrollToTop();
      }

      rootState.list = unionBy(rootState.list, list, "id");
    } finally {
      state.loading = false;

      if (state.pendingReload) {
        state.pendingReload = false;
        reload();
      }
    }
  };

  const reload = () => {
    const key = getQueryKey();

    state.queryKey = key;
    state.page = 1;
    state.noMore = false;

    rootState.list = [];
    rootState.activeId = void 0;

    if (state.loading) {
      state.pendingReload = true;
      return;
    }

    return fetchData(key, 1);
  };

  const loadMore = () => {
    if (state.noMore) return;
    if (state.loading) return;

    const nextPage = state.page + 1;

    state.page = nextPage;

    fetchData(state.queryKey || getQueryKey(), nextPage);
  };

  useTauriListen(LISTEN_KEY.REFRESH_CLIPBOARD_LIST, reload);

  useAsyncEffect(async () => {
    await reload();

    rootState.activeId = rootState.list[0]?.id;
  }, [rootState.group, rootState.search]);

  return {
    loadMore,
    reload,
  };
};
