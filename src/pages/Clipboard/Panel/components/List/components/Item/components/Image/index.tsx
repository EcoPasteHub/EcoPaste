import type { HistoryTablePayload } from "@/types/database";
import { convertFileSrc } from "@tauri-apps/api/core";
import type { FC } from "react";

const Image: FC<Partial<HistoryTablePayload>> = (props) => {
	const { value = "" } = props;

	const [src, setSrc] = useState("");

	useMount(async () => {
		const src = await convertFileSrc(value);

		setSrc(src);
	});

	return <img src={src} className="max-h-full" />;
};

export default memo(Image);
