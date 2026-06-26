import { Image } from "antd";
import type { FC } from "react";
import { useTranslation } from "react-i18next";
import type { PreferenceSetting } from "../../types/preferences";

interface SponsorQrControlProps {
  setting: PreferenceSetting;
}

interface ImagePreviewProps {
  alt: string;
  src: string;
}

/**
 * 关于页赞赏二维码展示：作为只读设置行的右侧内容。
 */
export const SponsorQrControl: FC<SponsorQrControlProps> = (props) => {
  const { setting } = props;
  const { t } = useTranslation("preferences");

  if (setting.control.type !== "sponsorQr") return null;

  return <ImagePreview alt={t("about.sponsorQrAlt")} src="/sponsor-qr.png" />;
};

/**
 * 行内图片预览控件：点击缩略图后使用 antd Image 预览层放大查看。
 */
const ImagePreview: FC<ImagePreviewProps> = (props) => {
  const { alt, src } = props;

  return (
    <Image
      alt={alt}
      className="rounded-1 object-contain"
      height={96}
      preview={{ src }}
      src={src}
      width={96}
    />
  );
};
