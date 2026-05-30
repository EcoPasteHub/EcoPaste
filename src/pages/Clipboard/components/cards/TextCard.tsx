import { cn } from "@heroui/styles";
import DOMPurify from "dompurify";
import { useMemo } from "react";

import type { ClipboardItem } from "@/types/clipboard";

import Highlight from "../Highlight";

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

const TextCard = ({
  item,
  keyword = "",
}: {
  item: ClipboardItem;
  keyword?: string;
}) => {
  const isHtml = item.subKind === "html";
  const preview = item.searchText ?? item.content;
  const isLinkLike = item.subKind === "url" || item.subKind === "email";

  return (
    <div className="flex items-start gap-2">
      {item.subKind === "color" ? (
        <div
          aria-hidden
          className="mt-0.5 size-5 shrink-0 rounded border border-separator"
          style={{ background: item.content }}
        />
      ) : null}
      <div className="min-w-0 flex-1">
        <div className="text-muted text-xs">{variantLabel(item)}</div>
        {isHtml ? (
          <HtmlPreview html={item.content} />
        ) : (
          <div
            className={cn("text-sm", {
              "line-clamp-2 break-all": !isLinkLike,
              "truncate text-link": isLinkLike,
            })}
          >
            <Highlight keyword={keyword} text={preview} />
          </div>
        )}
      </div>
    </div>
  );
};

export default TextCard;
