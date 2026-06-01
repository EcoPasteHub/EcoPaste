import { proxy } from "valtio";
import type { ClipboardGroup, ClipboardItemSort } from "@/types/clipboard";

interface ClipboardViewState {
  keyword: string;
  sort: ClipboardItemSort;
  group: ClipboardGroup;
}

/**
 * 剪贴板窗口的 UI 临时状态（非持久化）。
 * 跨组件共享：Header 搜索框写入 `keyword`，Group 写入 `group`，List 监听后驱动查询。
 * `limit` / `offset` 不在这里——分页由 `useClipboardItems` 内部 `useInfiniteScroll` 管理。
 */
export const clipboardViewState = proxy<ClipboardViewState>({
  group: "all",
  keyword: "",
  sort: "createdAtDesc",
});
