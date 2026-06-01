import { proxy } from "valtio";
import type { ClipboardItemSort } from "@/types/clipboard";

interface ClipboardViewState {
  keyword: string;
  sort: ClipboardItemSort;
}

/**
 * 剪贴板窗口的 UI 临时状态（非持久化）。
 * 跨组件共享：Header 搜索框写入 `keyword`，List 监听后驱动查询。
 * `limit` / `offset` 不在这里——分页由 `useClipboardItems` 内部 `useInfiniteScroll` 管理。
 */
export const clipboardViewState = proxy<ClipboardViewState>({
  keyword: "",
  sort: "createdAtDesc",
});
