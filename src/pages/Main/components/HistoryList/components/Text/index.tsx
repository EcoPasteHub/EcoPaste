import { Flex } from "antd";
import clsx from "clsx";
import { type CSSProperties, forwardRef, useContext } from "react";
import { Marker } from "react-mark.js";
import { useSnapshot } from "valtio";
import { MainContext } from "@/pages/Main";
import { clipboardStore } from "@/stores/clipboard";
import type { DatabaseSchemaHistory } from "@/types/database";

interface TextProps extends DatabaseSchemaHistory<"text"> {
  expanded?: boolean;
}

const Text = forwardRef<HTMLDivElement, TextProps>((props, ref) => {
  const { value, subtype, expanded } = props;
  const { rootState } = useContext(MainContext);
  const { content } = useSnapshot(clipboardStore);

  const displayLines = content.displayLines || 4;

  const renderMarker = () => {
    return <Marker mark={rootState.search}>{value}</Marker>;
  };

  const renderColor = () => {
    const className = "absolute rounded-full";
    const style: CSSProperties = {
      background: value,
    };

    return (
      <Flex align="center" gap="small">
        <div className="relative h-5.5 min-w-5.5">
          <span
            className={clsx(className, "inset-0 opacity-50")}
            style={style}
          />

          <span className={clsx(className, "inset-0.5")} style={style} />
        </div>

        {renderMarker()}
      </Flex>
    );
  };

  const renderContent = () => {
    if (subtype === "color") {
      return renderColor();
    }

    return renderMarker();
  };

  // 动态 line-clamp 样式
  const getLineClampStyle = (): CSSProperties => {
    if (expanded) {
      return {
        whiteSpace: "pre-wrap",
        wordBreak: "break-all",
      };
    }
    return {
      display: "-webkit-box",
      overflow: "hidden",
      WebkitBoxOrient: "vertical",
      WebkitLineClamp: displayLines,
      whiteSpace: "pre-wrap", // 确保换行符被尊重
      wordBreak: "break-all",
    };
  };

  return (
    <div ref={ref} style={getLineClampStyle()}>
      {renderContent()}
    </div>
  );
});

Text.displayName = "Text";

export default Text;
