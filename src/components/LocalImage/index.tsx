import { convertFileSrc } from "@tauri-apps/api/core";
import type { FC, HTMLAttributes } from "react";

interface LocalImage extends HTMLAttributes<HTMLImageElement> {
	src: string;
}

const LocalImage: FC<LocalImage> = (props) => {
	const { src, ...rest } = props;

	return <img {...rest} src={convertFileSrc(src)} />;
};

export default LocalImage;
