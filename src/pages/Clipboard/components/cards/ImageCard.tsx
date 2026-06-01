import type { FC } from "react";
import AssetImage from "@/components/AssetImage";
import type { ClipboardItem } from "@/types/clipboard";

/**
 * 图片类卡片：按 `content`（文件名）向 Rust 取缩略图路径并 `convertFileSrc` 加载。
 * 宽高来自 DB（width/height），不存在时不展示尺寸文案。
 */
const ImageCard: FC<ClipboardItem> = (props) => {
  const { imageThumbnailPath } = props;

  return <AssetImage className="max-h-21" src={imageThumbnailPath} />;
};

export default ImageCard;
