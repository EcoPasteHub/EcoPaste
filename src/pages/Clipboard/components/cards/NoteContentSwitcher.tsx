import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import type { FC, ReactNode } from "react";
import { useLayoutEffect, useRef, useState } from "react";
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
 * motion layout 负责高度变化时的过渡，高度不变时只做内容淡入淡出。
 */
const NoteContentSwitcher: FC<NoteContentSwitcherProps> = (props) => {
  const { children, note, showOriginal } = props;
  const shouldReduceMotion = useReducedMotion();
  const contentRef = useRef<HTMLDivElement>(null);
  const [height, setHeight] = useState<number | "auto">("auto");
  const activeKey = showOriginal ? "original" : "note";
  const transition = {
    duration: shouldReduceMotion ? 0 : 0.18,
    ease: "easeOut",
  } as const;

  useLayoutEffect(() => {
    const node = contentRef.current;

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
  });

  return (
    <motion.div
      animate={{ height }}
      className="overflow-hidden"
      initial={false}
      transition={transition}
    >
      <AnimatePresence initial={false} mode="popLayout">
        <motion.div
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          initial={{ opacity: 0 }}
          key={activeKey}
          ref={contentRef}
          transition={transition}
        >
          {showOriginal ? children : <NoteAnnotation note={note} />}
        </motion.div>
      </AnimatePresence>
    </motion.div>
  );
};

export default NoteContentSwitcher;
