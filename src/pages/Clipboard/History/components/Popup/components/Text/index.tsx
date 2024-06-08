import type { HistoryItem } from "@/types/database";
import type { FC } from "react";

const Text: FC<HistoryItem> = (props) => {
	const { content } = props;

	return content;
};

export default Text;
