import { motion } from "motion/react";
import type { FC, ReactNode } from "react";
import { PREVIEW_CONTENT_TRANSITION } from "../constants";

interface PreviewContentTransitionProps {
  children: ReactNode;
  contentKey: string;
}

/**
 * 内容切换时做轻量淡入位移，避免 payload 更换出现硬切。
 */
const PreviewContentTransition: FC<PreviewContentTransitionProps> = (props) => {
  const { children, contentKey } = props;

  return (
    <motion.div
      animate={{ opacity: 1, y: 0 }}
      className="min-h-full"
      initial={{ opacity: 0, y: 1 }}
      key={contentKey}
      transition={PREVIEW_CONTENT_TRANSITION}
    >
      {children}
    </motion.div>
  );
};

export default PreviewContentTransition;
