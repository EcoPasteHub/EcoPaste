import type { FC, ReactNode } from "react";

interface ControlFrameProps {
  children: ReactNode;
}

/**
 * 透传右侧控制内容，控件自身使用 antd 默认样式。
 */
const ControlFrame: FC<ControlFrameProps> = (props) => {
  const { children } = props;

  return <>{children}</>;
};

export default ControlFrame;
