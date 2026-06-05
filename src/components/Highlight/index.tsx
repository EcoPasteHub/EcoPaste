import type { FC, ReactNode } from "react";

interface HighlightProps {
  /**
   * 待渲染的纯文本。
   */
  text: string;
  /**
   * 搜索关键词，空串 / 仅空白时直接渲染原文。
   * 匹配规则与 Rust 后端一致：大小写不敏感子串匹配，不做正则 / 分词。
   */
  keyword?: string;
  /**
   * 透传到外层 `<span>` 的类名（外层不影响布局，仅作为可选样式容器）。
   */
  className?: string;
}

/**
 * 在 `text` 中把 `keyword` 命中的子串包成 `<mark>`，命中规则与后端 SQL
 * `LIKE '%kw%'` / FTS5 trigram 等价（大小写不敏感、按 Unicode 码元切分）。
 *
 * 设计为纯展示组件：不读取任何 store，keyword 由调用方注入，便于在非剪贴板
 * 场景复用。空 keyword 走快路径直接返回原文，零分配。
 */
const Highlight: FC<HighlightProps> = (props) => {
  const { text, keyword, className } = props;

  const kw = keyword?.trim() ?? "";

  if (!kw) return <span className={className}>{text}</span>;

  const lowerText = text.toLowerCase();
  const lowerKw = kw.toLowerCase();
  const kwLen = lowerKw.length;

  const nodes: ReactNode[] = [];
  let cursor = 0;
  let matchIndex = lowerText.indexOf(lowerKw);
  let key = 0;

  while (matchIndex !== -1) {
    if (matchIndex > cursor) {
      nodes.push(text.slice(cursor, matchIndex));
    }

    nodes.push(
      <mark className="bg-ant-gold-3" key={key++}>
        {text.slice(matchIndex, matchIndex + kwLen)}
      </mark>,
    );

    cursor = matchIndex + kwLen;
    matchIndex = lowerText.indexOf(lowerKw, cursor);
  }

  if (cursor < text.length) {
    nodes.push(text.slice(cursor));
  }

  return <span className={className}>{nodes}</span>;
};

export default Highlight;
