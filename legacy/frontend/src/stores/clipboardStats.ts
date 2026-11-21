import { proxy } from "valtio";

interface ClipboardStatsState {
  /**
   * 当前 List 查询过滤下的总条数。由 `List` 在每次 Rust 返回新 page 时回写，
   * `Footer` 订阅展示「共 N 项」，避免单独再发一次 IPC。
   * 未加载完成时为 `null`，Footer 据此渲染占位（如显示 0 或不显示）。
   */
  total: number | null;
}

/**
 * 列表统计共享态：跨 List/Footer 复用 Rust 列表查询附带的 `total`，
 * 切勿与 `clipboardViewState` 合并——后者会被 `...rest` 透传成查询参数。
 */
export const clipboardStatsState = proxy<ClipboardStatsState>({
  total: null,
});
