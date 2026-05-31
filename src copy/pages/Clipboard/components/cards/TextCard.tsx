import type { ClipboardItem } from "@/types/clipboard";
import { cn } from "@/utils/cn";

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
  // text 类型在列表查询里 content / searchText 都被置空，渲染只看 summary。
  // summary 为空仅在 image/files 等无意义场景出现，对 TextCard 不会发生。
  const preview = item.summary ?? "";
  const isLinkLike = item.subKind === "url" || item.subKind === "email";

  return (
    <div className="flex items-start gap-2">
      {item.subKind === "color" ? (
        <div
          aria-hidden
          className="b-split mt-0.5 size-5 shrink-0 rounded border"
          style={{ background: preview }}
        />
      ) : null}
      <div className="min-w-0 flex-1">
        <div className="c-text-tertiary text-xs">{variantLabel(item)}</div>
        <div
          className={cn("text-sm", {
            "c-link truncate": isLinkLike,
            "line-clamp-2 whitespace-pre-wrap break-all": !isLinkLike,
          })}
        >
          <Highlight keyword={keyword} text={preview} />
        </div>
      </div>
    </div>
  );
};

export default TextCard;
