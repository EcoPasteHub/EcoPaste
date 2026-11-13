import { AnimatePresence, motion } from "motion/react";
import type { FC, ReactNode } from "react";
import { PREVIEW_CONTENT_TRANSITION } from "../constants";

interface PreviewContentTransitionProps {
  children: ReactNode;
  contentKey: string;
}

/**
 * 内容切换时做轻量淡出 / 淡入位移，避免 payload 更换出现硬切。
 */
const PreviewContentTransition: FC<PreviewContentTransitionProps> = (props) => {
  const { children, contentKey } = props;

  return (
    <AnimatePresence initial={false} mode="wait">
      <motion.div
        animate={{ opacity: 1, y: 0 }}
        className="flex min-h-0 flex-1 flex-col"
        exit={{ opacity: 0, y: -1 }}
        initial={{ opacity: 0, y: 1 }}
        key={contentKey}
        transition={PREVIEW_CONTENT_TRANSITION}
      >
        {children}
      </motion.div>
    </AnimatePresence>
  );
};

export default PreviewContentTransition;
