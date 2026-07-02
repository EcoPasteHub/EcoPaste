import type { CSSProperties, FC } from "react";
import { useSnapshot } from "valtio";
import AssetImage from "@/components/AssetImage";
import { settingsState } from "@/stores/settings";
import type { ClipboardItem } from "@/types/clipboard";
import { cn } from "@/utils/cn";

interface ImageCardProps extends ClipboardItem {
  bottomSheet?: boolean;
}

/**
 * 图片类卡片：按 `content`（文件名）向 Rust 取缩略图路径并 `convertFileSrc` 加载。
 * 高度按设置限制，宽高来自 DB（width/height），不存在时不展示尺寸文案。
 */
const ImageCard: FC<ImageCardProps> = (props) => {
  const { bottomSheet = false, imageThumbnailPath } = props;
  const { clipboard } = useSnapshot(settingsState);
  const style: CSSProperties | undefined = bottomSheet
    ? void 0
    : {
        maxHeight: clipboard.display.imageMaxHeight,
      };

  return (
    <AssetImage
      className={cn({
        "h-full w-full rounded-2 object-cover": bottomSheet,
        "self-start": !bottomSheet,
      })}
      src={imageThumbnailPath}
      style={style}
    />
  );
};

export default ImageCard;
