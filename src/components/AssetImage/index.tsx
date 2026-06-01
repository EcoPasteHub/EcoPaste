import { convertFileSrc } from "@tauri-apps/api/core";
import type { FC, ImgHTMLAttributes } from "react";

interface AssetImageProps
  extends Omit<ImgHTMLAttributes<HTMLImageElement>, "src"> {
  src?: string | null;
  protocol?: string;
}

/**
 * 统一渲染 Tauri 本地文件图片：输入文件路径，内部转成 webview 可加载 URL。
 */
const AssetImage: FC<AssetImageProps> = (props) => {
  const { alt, protocol, src, ...rest } = props;

  if (!src) return null;

  return <img alt={alt} src={toAssetUrl(src, protocol)} {...rest} />;
};

/**
 * 把本地文件路径转为 webview 可访问地址；空路径返回空字符串以避免异常。
 */
const toAssetUrl = (filePath?: string | null, protocol?: string) => {
  if (!filePath) return "";

  if (!protocol) return convertFileSrc(filePath);

  return convertFileSrc(filePath, protocol);
};

export default AssetImage;
