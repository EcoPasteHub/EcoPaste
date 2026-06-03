import DOMPurify from "dompurify";
import type { FC } from "react";

interface SafeHtmlProps {
  className?: string;
  value: string;
}

/**
 * 安全渲染剪贴板 HTML 片段，禁止可自动播放/控制的高风险属性。
 */
const SafeHtml: FC<SafeHtmlProps> = (props) => {
  const { className, value } = props;

  return (
    <div
      className={className}
      dangerouslySetInnerHTML={{
        __html: DOMPurify.sanitize(value, {
          FORBID_ATTR: ["target", "controls", "autoplay", "autoPlay"],
        }),
      }}
    />
  );
};

export default SafeHtml;
