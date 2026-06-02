import type { CSSProperties, FC } from "react";
import { useSnapshot } from "valtio";
import Highlight from "@/components/Highlight";
import { clipboardViewState } from "@/stores/clipboardView";
import type { ClipboardItem } from "@/types/clipboard";
import { cn } from "@/utils/cn";

/**
 * 文本类卡片：渲染 summary（列表视图 content 已置空），最多三行。
 * 子类型（HTML/RTF/URL/Email/Color/Path）以小 Tag 提示。
 */
const TextCard: FC<ClipboardItem> = (props) => {
  const { summary, subKind, colorPreview } = props;
  const { keyword } = useSnapshot(clipboardViewState);

  if (subKind === "color" && colorPreview) {
    const style: CSSProperties = {
      background: colorPreview,
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

        {colorPreview}
      </div>
    );
  }

  return (
    <div className="line-clamp-3 whitespace-pre-wrap">
      <Highlight keyword={keyword} text={summary ?? ""} />
    </div>
  );
};

export default TextCard;
