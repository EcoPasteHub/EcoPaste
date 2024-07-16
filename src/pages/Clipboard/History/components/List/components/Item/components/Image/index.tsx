import type { HistoryItem } from "@/types/database";
import { convertFileSrc } from "@tauri-apps/api/tauri";
import type { FC } from "react";

const Image: FC<HistoryItem> = (props) => {
	const { value } = props;

	const [src, setSrc] = useState("");

	useMount(async () => {
		const src = await convertFileSrc(value);

		setSrc(src);
	});

	return <img src={src} />;
};

export default memo(Image);
