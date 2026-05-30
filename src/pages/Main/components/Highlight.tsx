// 关键词高亮：用 <mark> 包裹匹配片段。大小写不敏感；按搜索框原样切分（不分词），
// 与 Rust 端 FTS5 的前缀匹配近似——FTS 命中是否一定能在预览里 substring 命中并不严格，
// 因为 FTS 索引 `search_text`，预览也用 `search_text`（fallback 到 content）。命中不到时
// 整段原样渲染，不影响阅读。

const escapeRegExp = (s: string): string =>
  s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

interface Props {
  text: string;
  keyword: string;
}

const Highlight = ({ text, keyword }: Props) => {
  const kw = keyword.trim();
  if (kw.length === 0) return <>{text}</>;

  const re = new RegExp(`(${escapeRegExp(kw)})`, "gi");
  const parts = text.split(re);
  return (
    <>
      {parts.map((part, i) => {
        // 用 `index-part` 拼 key：split 结果索引稳定，同片段重复时 part 自身去同。
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
