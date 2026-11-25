import { isDev } from "@/utils/is";
import { log } from "@/utils/log";

/**
 * 「窗口可见 → 列表内容已刷新并绘制」这段耗时的临时探针（仅 dev 生效）。
 *
 * 打开窗口时会有两段可观测的时间：窗体出现（Rust `show_window`）和内容刷新（前端）。
 * 验收只看后者，所以这里以「前端收到 `window://visibility` 可见事件」为起点，
 * 以「刷新后的首个合成帧」为终点；用双 `requestAnimationFrame` 贴近实际绘制时刻。
 */

let shownAt = 0;
let pending = false;

/** 本次打开「确实需要刷新」的起点：收到可见事件且存在隐藏期挂起的刷新。 */
export const markClipboardWindowShown = () => {
  if (!isDev) return;
  shownAt = performance.now();
  pending = true;
};

/**
 * 本次打开触发的刷新已提交到 state 时调用。
 *
 * `source`：`reload`（打开时才去取数）/ `noop`（版本号一致，页面本就没有变化）/
 * `adopted`（隐藏期预热页已被采纳，内容早于可见就准备好）。
 */
export const settleClipboardOpen = (source: "reload" | "noop" | "adopted") => {
  if (!isDev || !pending) return;

  // 超过 1.5s 才落地的刷新已经与「这次打开」无关（例如打开时并未回顶、后来才滚动触发），
  // 记下来只会污染统计，直接丢弃。
  if (performance.now() - shownAt > 1500) {
    pending = false;
    return;
  }
  pending = false;

  const committedAt = performance.now();
  log.debug("clipboard open → content (commit)", {
    commitMs: Math.round((committedAt - shownAt) * 10) / 10,
    source,
  });

  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      log.debug("clipboard open → content (paint)", {
        paintMs: Math.round((performance.now() - shownAt) * 10) / 10,
        source,
      });
    });
  });
};
