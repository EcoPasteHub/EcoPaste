import type { HistoryItem } from "@/types/database";
import type { FC } from "react";

const Files: FC<HistoryItem> = (props) => {
	const { content = "" } = props;

	const paths: string[] = JSON.parse(content);

	return <div>{paths.join("\n")}</div>;
};

export default Files;
