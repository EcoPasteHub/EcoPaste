import { type CSSProperties, forwardRef } from "react";
import { useSnapshot } from "valtio";
import LocalImage from "@/components/LocalImage";
import { clipboardStore } from "@/stores/clipboard";
import type { DatabaseSchemaHistory } from "@/types/database";

interface ImageProps extends DatabaseSchemaHistory<"image"> {
  expanded?: boolean;
  onLoad?: () => void;
}

const Image = forwardRef<HTMLImageElement, ImageProps>((props, ref) => {
  const { value, expanded, onLoad } = props;
  const { content } = useSnapshot(clipboardStore);

  const imageDisplayHeight = content.imageDisplayHeight || 100;

  const getImageStyle = (): CSSProperties => {
    if (expanded) {
      return {
        maxHeight: "none",
        objectFit: "contain",
        width: "100%",
      };
    }
    return {
      maxHeight: `${imageDisplayHeight}px`,
      objectFit: "contain",
    };
  };

  return (
    <LocalImage onLoad={onLoad} ref={ref} src={value} style={getImageStyle()} />
  );
});

Image.displayName = "Image";

export default Image;
