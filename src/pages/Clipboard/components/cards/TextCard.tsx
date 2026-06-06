import type { CSSProperties, FC, MouseEvent } from "react";
import { useSnapshot } from "valtio";
import Highlight from "@/components/Highlight";
import { clipboardViewState } from "@/stores/clipboardView";
import { settingsState } from "@/stores/settings";
import type { ClipboardItem } from "@/types/clipboard";
import { cn } from "@/utils/cn";

interface TextCardProps extends ClipboardItem {
  /**
   * MOD 键按下时，URL / Email 以可点击链接样式渲染。
   */
  isLinkActive?: boolean;
  /**
   * 点击 URL / Email 文本时由列表层打开系统浏览器或邮件客户端。
   */
  onOpenLink?: () => void;
}

/**
 * 文本类卡片：渲染 summary（列表视图 content 已置空），按设置限制最大显示行数。
 * 子类型（HTML/RTF/URL/Email/Color/Path）以小 Tag 提示。
 */
const TextCard: FC<TextCardProps> = (props) => {
  const { summary, subKind, colorPreview, isLinkActive, onOpenLink } = props;
  const { keyword } = useSnapshot(clipboardViewState);
  const { clipboard } = useSnapshot(settingsState);
  const textMaxLines = clipboard.display.textMaxLines;
  const lineClampClass = textLineClampClass(textMaxLines);
  const isOpenableLink =
    isLinkActive && (subKind === "url" || subKind === "email");

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

  const handleLinkMouseDown = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
  };

  const handleLinkClick = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();

    onOpenLink?.();
  };

  if (isOpenableLink) {
    return (
      <button
        className={cn(
          "block w-full cursor-pointer whitespace-pre-wrap border-0 bg-transparent p-0 text-left text-ant-primary underline underline-offset-2",
          lineClampClass,
        )}
        onClick={handleLinkClick}
        onMouseDown={handleLinkMouseDown}
        type="button"
      >
        <Highlight keyword={keyword} text={summary ?? ""} />
      </button>
    );
  }

  return (
    <div className={cn("whitespace-pre-wrap", lineClampClass)}>
      <Highlight keyword={keyword} text={summary ?? ""} />
    </div>
  );
};

export default TextCard;

/**
 * 把用户设置夹到 UnoCSS safelist 覆盖的 line-clamp 类。
 */
function textLineClampClass(lines: number): string {
  if (lines <= 1) return "line-clamp-1";
  if (lines === 2) return "line-clamp-2";
  if (lines === 3) return "line-clamp-3";
  if (lines === 4) return "line-clamp-4";

  return "line-clamp-5";
}
