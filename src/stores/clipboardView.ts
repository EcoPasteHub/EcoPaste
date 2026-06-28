import { proxy } from "valtio";
import type { ClipboardCategory, ClipboardRange } from "@/types/clipboard";

interface ClipboardViewState {
  category: ClipboardCategory | null;
  keyword: string;
  groupId: string | null;
  range: ClipboardRange;
}

/**
 * 剪贴板窗口的 UI 临时状态（非持久化）。
 * 跨组件共享：Header 搜索框写入 `keyword`，Group 写入范围/分类/分组，List 监听后驱动查询。
 * 注意：这里的字段会被 List 用 `...rest` 透传成查询参数，**不要**塞进与 `ClipboardItemQuery` 同名
 * 但语义不同的字段（例如「窗口是否固定」要另起 store，否则会被当成 `pinned`(条目置顶) 过滤）。
 * `limit` / `offset` 不在这里——分页由 `useClipboardItems` 内部 `useInfiniteScroll` 管理。
 */
export const clipboardViewState = proxy<ClipboardViewState>({
  category: null,
  groupId: null,
  keyword: "",
  range: "all",
});
