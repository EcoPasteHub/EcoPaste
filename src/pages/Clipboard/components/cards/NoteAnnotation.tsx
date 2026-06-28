import type { FC } from "react";
import { useSnapshot } from "valtio";
import Highlight from "@/components/Highlight";
import { clipboardViewState } from "@/stores/clipboardView";

interface NoteAnnotationProps {
  /**
   * 已归一化的备注内容；调用方仅在非空时渲染本组件。
   */
  note: string;
}

/**
 * 剪贴板条目的内联备注批注：作为有备注条目的默认内容展示，并支持搜索高亮。
 */
const NoteAnnotation: FC<NoteAnnotationProps> = (props) => {
  const { note } = props;
  const { keyword } = useSnapshot(clipboardViewState);

  return (
    <div className="w-full whitespace-pre-wrap">
      <i
        aria-hidden="true"
        className="i-lucide:notebook-pen mr-0.5 inline-block size-3.5 translate-y-0.5 text-ant-primary"
      />

      <Highlight className="break-words" keyword={keyword} text={note} />
    </div>
  );
};

export default NoteAnnotation;
