import type { HistoryItem } from "@/types/database";
import type { FC } from "react";

const Html: FC<HistoryItem> = (props) => {
	const { content = "" } = props;

	return <div dangerouslySetInnerHTML={{ __html: content }} />;
};

export default Html;
