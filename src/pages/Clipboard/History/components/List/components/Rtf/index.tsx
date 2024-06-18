import type { HistoryItem } from "@/types/database";
import type { FC } from "react";

const Rtf: FC<HistoryItem> = (props) => {
	const { value } = props;

	return value;
};

export default Rtf;
