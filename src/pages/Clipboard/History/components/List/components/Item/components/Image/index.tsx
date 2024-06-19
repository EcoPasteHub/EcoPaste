import type { HistoryItem } from "@/types/database";
import { isNil } from "lodash-es";
import type { FC } from "react";

const Image: FC<HistoryItem> = (props) => {
	const { value = "" } = props;

	const [base64, setBase64] = useState("");

	useMount(async () => {
		let base64 = await getStore(value);

		if (isNil(base64)) {
			base64 = await getImageBase64(value);

			setStore(value, base64);
		}

		setBase64(base64);
	});

	return <img src={base64} />;
};

export default Image;
