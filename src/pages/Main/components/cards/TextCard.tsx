import DOMPurify from "dompurify";
import { useMemo } from "react";

import type { ClipboardItem } from "@/types/clipboard";

const KIND_LABEL: Record<string, string> = {
  color: "Color",
  email: "Email",
  html: "HTML",
  path: "Path",
  rtf: "RTF",
  text: "Text",
  url: "URL",
};

const variantLabel = (item: ClipboardItem): string => {
  if (item.subKind) return KIND_LABEL[item.subKind] ?? item.subKind;
  return KIND_LABEL[item.kind] ?? item.kind;
};

// 渲染富 HTML 预览：限定高度 + 折行，避免大段 HTML 撑爆列表行。
const HtmlPreview = ({ html }: { html: string }) => {
  const safe = useMemo(
    () => DOMPurify.sanitize(html, { FORBID_TAGS: ["script", "style"] }),
    [html],
  );
  return (
    <div
      className="prose prose-sm max-h-16 overflow-hidden break-all text-sm"
      dangerouslySetInnerHTML={{ __html: safe }}
    />
  );
};

const TextCard = ({ item }: { item: ClipboardItem }) => {
  const isHtml = item.subKind === "html";
  const preview = item.searchText ?? item.content;

  return (
    <div className="flex items-start gap-2">
      {item.subKind === "color" ? (
        <div
          aria-hidden
          className="mt-0.5 size-5 shrink-0 rounded border border-divider"
          style={{ background: item.content }}
        />
      ) : null}
      <div className="min-w-0 flex-1">
        <div className="text-default-500 text-xs">{variantLabel(item)}</div>
        {isHtml ? (
          <HtmlPreview html={item.content} />
        ) : (
          <div
            className={
              item.subKind === "url" || item.subKind === "email"
                ? "truncate text-primary text-sm"
                : "line-clamp-2 break-all text-sm"
            }
          >
            {preview}
          </div>
        )}
      </div>
    </div>
  );
};

export default TextCard;
