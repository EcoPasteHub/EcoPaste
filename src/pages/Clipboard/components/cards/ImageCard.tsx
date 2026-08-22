import type { CSSProperties, FC } from "react";
import { useSnapshot } from "valtio";
import AssetImage from "@/components/AssetImage";
import { settingsState } from "@/stores/settings";
import type { ClipboardItem } from "@/types/clipboard";
import { cn } from "@/utils/cn";

interface ImageCardProps extends ClipboardItem {
  /**
   * Fill the dock card instead of using the list thumbnail height cap.
   */
  fill?: boolean;
}

/**
 * 图片类卡片：按 `content`（文件名）向 Rust 取缩略图路径并 `convertFileSrc` 加载。
 * 高度按设置限制，宽高来自 DB（width/height），不存在时不展示尺寸文案。
 */
const ImageCard: FC<ImageCardProps> = (props) => {
  const { fill, imageThumbnailPath } = props;
  const { clipboard } = useSnapshot(settingsState);
  const style: CSSProperties | undefined = fill
    ? void 0
    : {
        maxHeight: clipboard.display.imageMaxHeight,
      };

  return (
    <AssetImage
      className={cn("self-start", {
        "h-full w-full self-stretch object-cover": fill,
      })}
      src={imageThumbnailPath}
      style={style}
    />
  );
};

export default ImageCard;
