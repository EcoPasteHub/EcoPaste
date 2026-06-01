import type { CSSProperties, FC } from "react";
import type { ClipboardItem } from "@/types/clipboard";
import { cn } from "@/utils/cn";

/**
 * 文本类卡片：渲染 summary（列表视图 content 已置空），最多三行。
 * 子类型（HTML/RTF/URL/Email/Color/Path）以小 Tag 提示。
 */
const TextCard: FC<ClipboardItem> = (props) => {
  const { summary, subKind } = props;

  if (subKind === "color") {
    const style: CSSProperties = {
      background: summary ?? void 0,
    };

    return (
      <div className="flex items-center gap-2">
        <div className="relative h-5.5 min-w-5.5">
          <span
            className={"absolute inset-0 rounded-full opacity-50"}
            style={style}
          />

          <span
            className={cn("absolute inset-0.5 rounded-full")}
            style={style}
          />
        </div>

        {summary}
      </div>
    );
  }

  return <div className="line-clamp-3 whitespace-pre-wrap">{summary}</div>;
};

export default TextCard;
