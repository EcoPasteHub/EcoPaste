import type { HistoryItem } from "@/types/database";
import type { FC } from "react";

const Files: FC<HistoryItem> = (props) => {
	const { content = "" } = props;

	const a: string[] = JSON.parse(content);

	return <div>{a.join("\n")}</div>;
};

export default Files;
