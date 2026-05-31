import { cn } from "@heroui/styles";

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

const TextCard = ({
  item,
  keyword = "",
}: {
  item: ClipboardItem;
  keyword?: string;
}) => {
  // HTML/RTF/纯文本都用 summary（Rust 入库时按 512 字符截断的纯文本）；
  // 列表不再渲染原始 HTML/RTF，避免大段富文本撑爆 DOM。
  // summary 为 null 时（旧数据兜底）退回到 searchText / content。
  const preview = item.summary ?? item.searchText ?? item.content;
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
        <div
          className={cn("text-sm", {
            "line-clamp-2 break-all": !isLinkLike,
            "truncate text-link": isLinkLike,
          })}
        >
          <Highlight keyword={keyword} text={preview} />
        </div>
      </div>
    </div>
  );
};

export default TextCard;
