import type { HistoryItem } from "@/types/database";
import type { FC } from "react";

const Image: FC<HistoryItem> = (props) => {
	const { value = "" } = props;

	return <img src={value} />;
};

export default Image;
