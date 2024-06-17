import type { HistoryItem } from "@/types/database";
import type { FC } from "react";

const Image: FC<HistoryItem> = (props) => {
	const { value = "" } = props;

	const [base64, setBase64] = useState("");

	useMount(async () => {
		const base64 = await getImageBase64(value);

		setBase64(base64);
	});

	return <img src={base64} />;
};

export default Image;
