import { motion, useReducedMotion } from "motion/react";
import type { FC, ReactNode } from "react";
import { useLayoutEffect, useRef, useState } from "react";
import { cn } from "@/utils/cn";
import NoteAnnotation from "./NoteAnnotation";

interface NoteContentSwitcherProps {
  /**
   * 备注内容；默认态展示它，hover 时切换为原始剪贴板内容。
   */
  note: string;
  /**
   * 原始剪贴板内容，由具体 kind 卡片渲染完成后注入。
   */
  children: ReactNode;
  /**
   * 是否展示原始内容；由整张卡片 hover 状态驱动。
   */
  showOriginal: boolean;
}

/**
 * 有备注条目的内容切换器：默认显示备注，鼠标进入内容区时预览原始内容；
 * 备注和原内容常驻 DOM，仅测量当前可见层，避免虚拟列表在 hover
 * 切换瞬间读到卸载后的空内容高度。
 */
const NoteContentSwitcher: FC<NoteContentSwitcherProps> = (props) => {
  const { children, note, showOriginal } = props;
  const shouldReduceMotion = useReducedMotion();
  const noteRef = useRef<HTMLDivElement>(null);
  const originalRef = useRef<HTMLDivElement>(null);
  const [height, setHeight] = useState<number | "auto">("auto");
  const transition = {
    duration: shouldReduceMotion ? 0 : 0.18,
    ease: "easeOut",
  } as const;

  useLayoutEffect(() => {
    const node = showOriginal ? originalRef.current : noteRef.current;

    if (!node) return;

    const updateHeight = () => {
      const nextHeight = node.getBoundingClientRect().height;

      setHeight((current) => {
        return current === nextHeight ? current : nextHeight;
      });
    };

    updateHeight();

    if (typeof ResizeObserver === "undefined") return;

    const observer = new ResizeObserver(updateHeight);
    observer.observe(node);

    return () => {
      observer.disconnect();
    };
  }, [showOriginal]);

  return (
    <motion.div
      animate={{ height }}
      className="relative overflow-hidden"
      initial={false}
      transition={transition}
    >
      <motion.div
        animate={{ opacity: showOriginal ? 0 : 1 }}
        aria-hidden={showOriginal}
        className={cn("absolute inset-x-0 top-0", {
          "pointer-events-none invisible": showOriginal,
        })}
        initial={false}
        ref={noteRef}
        transition={transition}
      >
        <NoteAnnotation note={note} />
      </motion.div>

      <motion.div
        animate={{ opacity: showOriginal ? 1 : 0 }}
        aria-hidden={!showOriginal}
        className={cn("absolute inset-x-0 top-0", {
          "pointer-events-none invisible": !showOriginal,
        })}
        initial={false}
        ref={originalRef}
        transition={transition}
      >
        {children}
      </motion.div>
    </motion.div>
  );
};

export default NoteContentSwitcher;
