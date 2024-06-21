import type { HistoryItem } from "@/types/database";
import type { FC } from "react";

const HTML: FC<HistoryItem> = (props) => {
	const { value = "" } = props;

	return <div dangerouslySetInnerHTML={{ __html: value }} />;
};

export default memo(HTML);
