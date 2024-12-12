import type { HistoryTablePayload } from "@/types/database";
import { convertFileSrc } from "@tauri-apps/api/core";
import type { FC } from "react";

interface ImageProps extends Partial<HistoryTablePayload> {
	className?: string;
}

const Image: FC<ImageProps> = (props) => {
	const { value = "", className = "max-h-full" } = props;

	const [src, setSrc] = useState("");

	useAsyncEffect(async () => {
		if (!value) return;

		const src = await convertFileSrc(value);

		setSrc(src);
	}, [value]);

	return <img src={src} className={className} />;
};

export default memo(Image);
