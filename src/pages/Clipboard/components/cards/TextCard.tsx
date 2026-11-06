import type { CSSProperties, FC, MouseEvent } from "react";
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "valtio";
import Highlight from "@/components/Highlight";
import Tooltip from "@/components/Tooltip";
import { clipboardViewState } from "@/stores/clipboardView";
import { settingsState } from "@/stores/settings";
import type { ClipboardItem } from "@/types/clipboard";
import { cn } from "@/utils/cn";
import { log } from "@/utils/log";

interface TextCardProps extends ClipboardItem {
  /**
   * MOD 键按下时，URL / Email 以可点击链接样式渲染。
   */
  isLinkActive?: boolean;
  /**
   * 点击 URL / Email 文本时由列表层打开系统浏览器或邮件客户端。
   */
  onOpenLink?: () => void;
  /**
   * 普通文本点击时复用卡片原有的单击处理；拖选时不调用，避免误粘贴或误复制。
   */
  onCardMouseDown?: (event: MouseEvent<HTMLDivElement>) => void;
}

interface SelectionCopyState {
  position: SelectionCopyPosition;
  text: string;
}

interface SelectionCopyPosition {
  left: number;
  top: number;
}

const COPY_BUTTON_SIZE = 20;
const FLOATING_BUTTON_GAP = 8;
const VIEWPORT_PADDING = 8;

/**
 * 文本类卡片：渲染 summary（列表视图 content 已置空），按设置限制最大显示行数。
 * 子类型（HTML/RTF/URL/Email/Color/Path）以小 Tag 提示。
 */
const TextCard: FC<TextCardProps> = (props) => {
  const {
    summary,
    subKind,
    colorPreview,
    isLinkActive,
    onCardMouseDown,
    onOpenLink,
  } = props;
  const { t } = useTranslation("clipboard");
  const { keyword } = useSnapshot(clipboardViewState);
  const { clipboard } = useSnapshot(settingsState);
  const textRootRef = useRef<HTMLDivElement | null>(null);
  const selectionGestureRef = useRef<AbortController | null>(null);
  const copiedResetTimerRef = useRef<number | null>(null);
  const [expanded, setExpanded] = useState(false);
  const [isTruncated, setIsTruncated] = useState(false);
  const [selectionCopy, setSelectionCopy] = useState<SelectionCopyState | null>(
    null,
  );
  const [copied, setCopied] = useState(false);
  const textMaxLines = clipboard.display.textMaxLines;
  const lineClampClass = textLineClampClass(textMaxLines);
  const color = subKind === "color" ? colorPreview : null;
  const selectionCopyVisible = selectionCopy !== null;
  const isOpenableLink =
    !selectionCopyVisible &&
    isLinkActive &&
    (subKind === "url" || subKind === "email");
  const text = summary ?? "";

  // biome-ignore lint/correctness/useExhaustiveDependencies: text and clamp changes require a fresh DOM overflow measurement.
  useEffect(() => {
    if (expanded) return;

    const textRoot = textRootRef.current;
    if (!textRoot) return;

    const updateTruncation = () => {
      const nextIsTruncated =
        textRoot.scrollHeight > textRoot.clientHeight ||
        textRoot.scrollWidth > textRoot.clientWidth;

      setIsTruncated((currentIsTruncated) => {
        return currentIsTruncated === nextIsTruncated
          ? currentIsTruncated
          : nextIsTruncated;
      });
    };

    updateTruncation();

    const observer = new ResizeObserver(updateTruncation);
    observer.observe(textRoot);

    return () => {
      observer.disconnect();
    };
  }, [expanded, text, textMaxLines]);

  useEffect(() => {
    if (!selectionCopyVisible) return;

    const handleSelectionChange = () => {
      const nextSelectionCopy = getSelectionCopyState(textRootRef.current);

      setSelectionCopy(nextSelectionCopy);

      if (nextSelectionCopy === null) setCopied(false);
    };

    document.addEventListener("selectionchange", handleSelectionChange);
    window.addEventListener("resize", handleSelectionChange);
    document.addEventListener("scroll", handleSelectionChange, true);

    return () => {
      document.removeEventListener("selectionchange", handleSelectionChange);
      window.removeEventListener("resize", handleSelectionChange);
      document.removeEventListener("scroll", handleSelectionChange, true);
    };
  }, [selectionCopyVisible]);

  useEffect(() => {
    return () => {
      selectionGestureRef.current?.abort();
      if (copiedResetTimerRef.current === null) return;

      window.clearTimeout(copiedResetTimerRef.current);
    };
  }, []);

  const stopCardInteraction = (event: MouseEvent<HTMLElement>) => {
    event.preventDefault();
    event.stopPropagation();
  };

  const handleTextMouseDown = (event: MouseEvent<HTMLDivElement>) => {
    if (event.button !== 0) return;

    event.stopPropagation();
    selectionGestureRef.current?.abort();

    const gesture = new AbortController();
    selectionGestureRef.current = gesture;

    // The card is draggable for OS drag-out, but text starts a selection.
    const card = event.currentTarget.closest<HTMLElement>("[draggable=true]");
    if (card) {
      card.draggable = false;
      gesture.signal.addEventListener("abort", () => {
        card.draggable = true;
      });
    }

    window.getSelection()?.removeAllRanges();

    const finishSelection = () => {
      gesture.abort();
      const nextSelectionCopy = getSelectionCopyState(textRootRef.current);
      if (!nextSelectionCopy) window.getSelection()?.removeAllRanges();

      setSelectionCopy(nextSelectionCopy);
      setCopied(false);
    };

    document.addEventListener("mouseup", finishSelection, {
      once: true,
      signal: gesture.signal,
    });
    window.addEventListener("blur", () => gesture.abort(), {
      once: true,
      signal: gesture.signal,
    });
  };

  const handleTextClick = (event: MouseEvent<HTMLDivElement>) => {
    if (!window.getSelection()?.isCollapsed) return;

    onCardMouseDown?.(event);
  };

  const handleLinkClick = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();

    if (getSelectionCopyState(textRootRef.current) !== null) return;

    onOpenLink?.();
  };

  const handleExpandToggle = (event: MouseEvent<HTMLButtonElement>) => {
    stopCardInteraction(event);
    if (selectionCopyVisible) window.getSelection()?.removeAllRanges();
    setExpanded((currentExpanded) => {
      return !currentExpanded;
    });
  };

  const handleSelectionCopy = async (event: MouseEvent<HTMLButtonElement>) => {
    stopCardInteraction(event);

    const currentSelection = getSelectionCopyState(textRootRef.current);
    if (!currentSelection) return;

    try {
      await navigator.clipboard.writeText(currentSelection.text);
      setCopied(true);

      if (copiedResetTimerRef.current !== null) {
        window.clearTimeout(copiedResetTimerRef.current);
      }

      copiedResetTimerRef.current = window.setTimeout(() => {
        setCopied(false);
        copiedResetTimerRef.current = null;
      }, 1000);
    } catch (error) {
      log.warn("Failed to copy selected clipboard text", error);
    }
  };

  const selectionCopyStyle: CSSProperties | undefined = selectionCopy
    ? {
        left: selectionCopy.position.left,
        top: selectionCopy.position.top,
      }
    : void 0;
  const selectionCopyLabel = copied
    ? t("textCard.copied")
    : t("textCard.copySelection");

  return (
    <>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: text selection delegates activation to the keyboard-accessible parent list. */}
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: the parent list owns keyboard navigation and activation. */}
      <div
        className={cn("select-text whitespace-pre-wrap", {
          [lineClampClass]: !expanded && !color,
          "break-words": expanded,
          "flex items-center gap-2 text-sm": color,
        })}
        data-clipboard-text
        draggable={false}
        onClick={handleTextClick}
        onMouseDown={handleTextMouseDown}
        ref={textRootRef}
      >
        {color ? (
          <>
            <span
              aria-hidden="true"
              className="size-4.5 shrink-0 rounded-1 border border-ant-border-secondary"
              style={{ background: color }}
            />
            <span className="font-mono leading-5">{color}</span>
          </>
        ) : isOpenableLink ? (
          <button
            className="block w-full cursor-pointer border-0 bg-transparent p-0 text-left text-ant-primary underline underline-offset-2"
            onClick={handleLinkClick}
            type="button"
          >
            <Highlight keyword={keyword} text={text} />
          </button>
        ) : (
          <Highlight keyword={keyword} text={text} />
        )}
      </div>

      {isTruncated ? (
        <div className="flex justify-end">
          <button
            className="rounded-1 border-0 bg-transparent px-1 py-0.5 text-ant-secondary text-xs transition-colors hover:bg-ant-fill-tertiary hover:text-ant-text motion-reduce:transition-none"
            onClick={handleExpandToggle}
            onContextMenu={stopCardInteraction}
            onDoubleClick={stopCardInteraction}
            onMouseDown={stopCardInteraction}
            type="button"
          >
            {expanded ? t("textCard.collapse") : t("textCard.expand")}
          </button>
        </div>
      ) : null}

      {selectionCopy
        ? createPortal(
            <Tooltip title={selectionCopyLabel}>
              <button
                aria-label={selectionCopyLabel}
                className="fixed z-20 flex size-5 items-center justify-center rounded-1.5 border-0 bg-ant-container text-ant-secondary shadow-sm transition-colors hover:bg-ant-fill-tertiary hover:text-ant-text motion-reduce:transition-none"
                onAuxClick={stopCardInteraction}
                onClick={handleSelectionCopy}
                onContextMenu={stopCardInteraction}
                onDoubleClick={stopCardInteraction}
                onMouseDown={stopCardInteraction}
                style={selectionCopyStyle}
                type="button"
              >
                <i
                  aria-hidden="true"
                  className={cn(
                    copied ? "i-lucide:check" : "i-lucide:copy",
                    "text-sm",
                  )}
                />
              </button>
            </Tooltip>,
            document.body,
          )
        : null}
    </>
  );
};

export default TextCard;

/**
 * 仅接受起止点都位于当前文本根节点内的浏览器选区，避免复制到其他卡片或页面区域。
 */
function getSelectionCopyState(
  textRoot: HTMLDivElement | null,
): SelectionCopyState | null {
  const selection = window.getSelection();

  if (
    !textRoot ||
    !selection ||
    selection.isCollapsed ||
    selection.rangeCount === 0
  ) {
    return null;
  }

  const text = selection.toString();
  const range = selection.getRangeAt(0);

  if (
    text.length === 0 ||
    !textRoot.contains(range.startContainer) ||
    !textRoot.contains(range.endContainer)
  ) {
    return null;
  }

  const rect = range.getBoundingClientRect();

  if (rect.width === 0 && rect.height === 0) return null;

  return {
    position: getSelectionCopyPosition(rect),
    text,
  };
}

/**
 * 将复制按钮放在选区旁边；空间不足时改置于选区上下，始终留在 WebView 可见区域。
 */
function getSelectionCopyPosition(rect: DOMRect): SelectionCopyPosition {
  const horizontalMaximum = Math.max(
    VIEWPORT_PADDING,
    window.innerWidth - VIEWPORT_PADDING - COPY_BUTTON_SIZE,
  );
  const verticalMaximum = Math.max(
    VIEWPORT_PADDING,
    window.innerHeight - VIEWPORT_PADDING - COPY_BUTTON_SIZE,
  );
  const canPlaceRight =
    rect.right + FLOATING_BUTTON_GAP + COPY_BUTTON_SIZE <=
    window.innerWidth - VIEWPORT_PADDING;
  const canPlaceLeft =
    rect.left - FLOATING_BUTTON_GAP - COPY_BUTTON_SIZE >= VIEWPORT_PADDING;

  if (canPlaceRight || canPlaceLeft) {
    return {
      left: canPlaceRight
        ? rect.right + FLOATING_BUTTON_GAP
        : rect.left - FLOATING_BUTTON_GAP - COPY_BUTTON_SIZE,
      top: clamp(
        rect.bottom - COPY_BUTTON_SIZE,
        VIEWPORT_PADDING,
        verticalMaximum,
      ),
    };
  }

  const top =
    rect.top - FLOATING_BUTTON_GAP - COPY_BUTTON_SIZE >= VIEWPORT_PADDING
      ? rect.top - FLOATING_BUTTON_GAP - COPY_BUTTON_SIZE
      : rect.bottom + FLOATING_BUTTON_GAP;

  return {
    left: clamp(rect.left, VIEWPORT_PADDING, horizontalMaximum),
    top: clamp(top, VIEWPORT_PADDING, verticalMaximum),
  };
}

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

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(Math.max(value, minimum), maximum);
}
