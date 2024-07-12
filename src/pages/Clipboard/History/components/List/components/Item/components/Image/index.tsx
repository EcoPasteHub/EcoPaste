import type { HistoryItem } from "@/types/database";
import { convertFileSrc } from "@tauri-apps/api/tauri";
import type { FC } from "react";

const Image: FC<HistoryItem> = (props) => {
	const { id, value = "" } = props;

	const [src, setSrc] = useState("");

	useMount(async () => {
		const saveImageDir = await getSaveImageDir();

		let path = value;

		if (value.includes(saveImageDir)) {
			// 为了适配导出功能，把旧图片路径替换为文件名
			updateSQL("history", {
				id,
				value: value.replace(saveImageDir, ""),
			});
		} else {
			path = saveImageDir + path;
		}

		const src = await convertFileSrc(path);

		setSrc(src);
	});

	return <img src={src} />;
};

export default memo(Image);
