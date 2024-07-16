import type { HistoryItem } from "@/types/database";
import { convertFileSrc } from "@tauri-apps/api/tauri";
import type { FC } from "react";

const Image: FC<HistoryItem> = (props) => {
	const { value } = props;

	const [src, setSrc] = useState("");

	useMount(async () => {
		const saveImageDir = await getSaveImageDir();

		const src = await convertFileSrc(saveImageDir + value);

		setSrc(src);
	});

	return <img src={src} />;
};

export default memo(Image);
