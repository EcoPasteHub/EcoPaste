const escapeRegExp = (s: string): string =>
  s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

interface Props {
  text: string;
  keyword: string;
}

/**
 * 关键词高亮：用 <mark> 包裹匹配片段。大小写不敏感；按搜索框原样切分（不分词）。
 * 与 Rust 端 FTS5 前缀匹配近似但不严格——命中不到时整段原样渲染，不影响阅读。
 */
const Highlight = ({ text, keyword }: Props) => {
  const kw = keyword.trim();
  if (kw.length === 0) return <>{text}</>;

  const re = new RegExp(`(${escapeRegExp(kw)})`, "gi");
  const parts = text.split(re);
  return (
    <>
      {parts.map((part, i) => {
        const key = `${i}-${part}`;
        return i % 2 === 1 ? (
          <mark
            className="rounded-sm bg-warning-soft px-0.5 text-warning-soft-foreground"
            key={key}
          >
            {part}
          </mark>
        ) : (
          <span key={key}>{part}</span>
        );
      })}
    </>
  );
};

export default Highlight;
