import type { HistoryItem } from "@/types/database";
import type { FC } from "react";

const Rtf: FC<HistoryItem> = (props) => {
	const { content } = props;

	return content;
};

export default Rtf;
