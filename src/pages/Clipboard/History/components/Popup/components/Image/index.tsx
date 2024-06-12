import type { HistoryItem } from "@/types/database";
import type { FC } from "react";

const Image: FC<HistoryItem> = (props) => {
	const { content } = props;

	return <img src={`data:image/png;base64, ${content}`} />;
};

export default Image;
