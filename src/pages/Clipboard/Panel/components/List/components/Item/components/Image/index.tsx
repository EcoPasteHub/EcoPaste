import type { ClipboardItem } from "@/types/database";
import { convertFileSrc } from "@tauri-apps/api/tauri";
import type { FC } from "react";

const Image: FC<Partial<ClipboardItem>> = (props) => {
	const { value = "" } = props;

	const [src, setSrc] = useState("");

	useMount(async () => {
		const src = await convertFileSrc(value);

		setSrc(src);
	});

	return <img src={src} className="max-h-full" />;
};

export default memo(Image);
